package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/config"
	"github.com/gophercloud/gophercloud/v2/openstack/config/clouds"
	bsquotas "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/quotasets"
	computequotas "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/quotasets"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

type ProjectInfo struct {
	ID   string
	Name string
}

type OpenStack struct {
	Provider     *gophercloud.ProviderClient
	EndpointOpts gophercloud.EndpointOpts
	CloudName    string
	UserName     string
	Region       string
	ProjectID    string
	ProjectName  string
	Projects     []ProjectInfo

	// Stored for re-auth on project switch
	origAuthOpts gophercloud.AuthOptions
	tlsConfig    *tls.Config
	insecure     bool

	compute      *gophercloud.ServiceClient
	network      *gophercloud.ServiceClient
	blockStorage *gophercloud.ServiceClient
	imageService *gophercloud.ServiceClient
	identity     *gophercloud.ServiceClient
	metric       *gophercloud.ServiceClient
	loadBalancer *gophercloud.ServiceClient
	dns          *gophercloud.ServiceClient
}

func New(ctx context.Context, cloudName string, insecure bool) (*OpenStack, error) {
	c := &OpenStack{CloudName: cloudName, insecure: insecure}
	if err := c.authenticate(ctx); err != nil {
		return nil, err
	}
	c.detectScopedProject()
	if err := c.loadProjects(ctx); err != nil {
		// Non-admin users may lack identity:list_projects permission.
		// Fall back to just the current scoped project.
		c.Projects = []ProjectInfo{{ID: c.ProjectID, Name: c.ProjectName}}
	}
	return c, nil
}

func (c *OpenStack) authenticate(ctx context.Context) error {
	c.clearClients()

	if c.CloudName != "" && c.CloudName != "env" {
		return c.authFromClouds(ctx)
	}
	return c.authFromEnv(ctx)
}

func (c *OpenStack) authFromClouds(ctx context.Context) error {
	authOpts, endpointOpts, tlsConfig, err := clouds.Parse(clouds.WithCloudName(c.CloudName))
	if err != nil {
		return fmt.Errorf("parsing clouds.yaml for %q: %w", c.CloudName, err)
	}
	authOpts.AllowReauth = true
	c.EndpointOpts = endpointOpts
	c.origAuthOpts = authOpts
	if c.insecure {
		if tlsConfig == nil {
			tlsConfig = &tls.Config{}
		}
		tlsConfig.InsecureSkipVerify = true
	}
	c.tlsConfig = tlsConfig

	provider, err := config.NewProviderClient(ctx, authOpts, config.WithTLSConfig(tlsConfig))
	if err != nil {
		return fmt.Errorf("authenticating cloud %q: %w", c.CloudName, err)
	}
	c.Provider = provider
	c.extractAuthInfo(authOpts)
	if c.Region == "" {
		c.Region = endpointOpts.Region
	}
	return nil
}

func (c *OpenStack) authFromEnv(ctx context.Context) error {
	authOpts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		return fmt.Errorf("reading OS_* env vars: %w", err)
	}
	authOpts.AllowReauth = true

	// openrc files typically set OS_USER_DOMAIN_NAME / OS_PROJECT_DOMAIN_NAME
	// but AuthOptionsFromEnv only reads OS_DOMAIN_ID / OS_DOMAIN_NAME.
	if authOpts.DomainID == "" && authOpts.DomainName == "" {
		if v := os.Getenv("OS_USER_DOMAIN_NAME"); v != "" {
			authOpts.DomainName = v
		} else if v := os.Getenv("OS_USER_DOMAIN_ID"); v != "" {
			authOpts.DomainID = v
		} else if v := os.Getenv("OS_PROJECT_DOMAIN_NAME"); v != "" {
			authOpts.DomainName = v
		} else if v := os.Getenv("OS_PROJECT_DOMAIN_ID"); v != "" {
			authOpts.DomainID = v
		}
	}

	c.origAuthOpts = authOpts

	var provider *gophercloud.ProviderClient
	if c.insecure {
		c.tlsConfig = &tls.Config{InsecureSkipVerify: true}
		provider, err = config.NewProviderClient(ctx, authOpts, config.WithTLSConfig(c.tlsConfig))
	} else {
		provider, err = openstack.AuthenticatedClient(ctx, authOpts)
	}
	if err != nil {
		return fmt.Errorf("authenticating from env: %w", err)
	}
	c.Provider = provider
	c.extractAuthInfo(authOpts)
	c.Region = os.Getenv("OS_REGION_NAME")
	return nil
}

func (c *OpenStack) extractAuthInfo(opts gophercloud.AuthOptions) {
	c.UserName = opts.Username
	c.ProjectID = opts.TenantID
	c.ProjectName = opts.TenantName
	if c.CloudName == "" {
		c.CloudName = "env"
	}
}

// detectScopedProject extracts the actual project from the auth token,
// which is needed when clouds.yaml uses project name instead of ID.
func (c *OpenStack) detectScopedProject() {
	authResult := c.Provider.GetAuthResult()
	if authResult == nil {
		return
	}
	if r, ok := authResult.(tokens.CreateResult); ok {
		project, err := r.ExtractProject()
		if err == nil && project != nil {
			c.ProjectID = project.ID
			c.ProjectName = project.Name
		}
	}
}

func (c *OpenStack) loadProjects(ctx context.Context) error {
	identityClient, err := c.Identity()
	if err != nil {
		return err
	}

	allPages, err := projects.List(identityClient, projects.ListOpts{}).AllPages(ctx)
	if err != nil {
		return fmt.Errorf("listing projects: %w", err)
	}
	allProjects, err := projects.ExtractProjects(allPages)
	if err != nil {
		return err
	}
	c.Projects = make([]ProjectInfo, 0, len(allProjects))
	for _, p := range allProjects {
		c.Projects = append(c.Projects, ProjectInfo{ID: p.ID, Name: p.Name})
	}
	return nil
}

func (c *OpenStack) SwitchProject(ctx context.Context, projectID, projectName string) error {
	c.clearClients()

	// Re-auth with original credentials scoped to the new project
	authOpts := c.origAuthOpts
	authOpts.TenantID = projectID
	authOpts.TenantName = ""
	authOpts.AllowReauth = true

	var provider *gophercloud.ProviderClient
	var err error

	if c.tlsConfig != nil {
		provider, err = config.NewProviderClient(ctx, authOpts, config.WithTLSConfig(c.tlsConfig))
	} else {
		provider, err = openstack.AuthenticatedClient(ctx, authOpts)
	}
	if err != nil {
		return fmt.Errorf("switching to project %s: %w", projectName, err)
	}

	c.Provider = provider
	c.ProjectID = projectID
	c.ProjectName = projectName

	return nil
}

func (c *OpenStack) clearClients() {
	c.compute = nil
	c.network = nil
	c.blockStorage = nil
	c.imageService = nil
	c.identity = nil
	c.metric = nil
	c.loadBalancer = nil
	c.dns = nil
}

func (c *OpenStack) Compute() (*gophercloud.ServiceClient, error) {
	if c.compute != nil {
		return c.compute, nil
	}
	client, err := openstack.NewComputeV2(context.Background(), c.Provider, c.EndpointOpts)
	if err != nil {
		return nil, fmt.Errorf("creating compute client: %w", err)
	}
	c.compute = client
	return c.compute, nil
}

func (c *OpenStack) Network() (*gophercloud.ServiceClient, error) {
	if c.network != nil {
		return c.network, nil
	}
	client, err := openstack.NewNetworkV2(context.Background(), c.Provider, c.EndpointOpts)
	if err != nil {
		return nil, fmt.Errorf("creating network client: %w", err)
	}
	c.network = client
	return c.network, nil
}

func (c *OpenStack) BlockStorage() (*gophercloud.ServiceClient, error) {
	if c.blockStorage != nil {
		return c.blockStorage, nil
	}
	client, err := openstack.NewBlockStorageV3(context.Background(), c.Provider, c.EndpointOpts)
	if err != nil {
		return nil, fmt.Errorf("creating block storage client: %w", err)
	}
	c.blockStorage = client
	return c.blockStorage, nil
}

func (c *OpenStack) ImageService() (*gophercloud.ServiceClient, error) {
	if c.imageService != nil {
		return c.imageService, nil
	}
	client, err := openstack.NewImageV2(context.Background(), c.Provider, c.EndpointOpts)
	if err != nil {
		return nil, fmt.Errorf("creating image client: %w", err)
	}
	c.imageService = client
	return c.imageService, nil
}

func (c *OpenStack) Identity() (*gophercloud.ServiceClient, error) {
	if c.identity != nil {
		return c.identity, nil
	}
	client, err := openstack.NewIdentityV3(context.Background(), c.Provider, c.EndpointOpts)
	if err != nil {
		return nil, fmt.Errorf("creating identity client: %w", err)
	}
	c.identity = client
	return c.identity, nil
}

func (c *OpenStack) LoadBalancer() (*gophercloud.ServiceClient, error) {
	if c.loadBalancer != nil {
		return c.loadBalancer, nil
	}
	client, err := openstack.NewLoadBalancerV2(context.Background(), c.Provider, c.EndpointOpts)
	if err != nil {
		return nil, fmt.Errorf("creating load balancer client: %w", err)
	}
	c.loadBalancer = client
	return c.loadBalancer, nil
}

func (c *OpenStack) DNS() (*gophercloud.ServiceClient, error) {
	if c.dns != nil {
		return c.dns, nil
	}
	client, err := openstack.NewDNSV2(context.Background(), c.Provider, c.EndpointOpts)
	if err != nil {
		return nil, fmt.Errorf("creating dns client: %w", err)
	}
	c.dns = client
	return c.dns, nil
}

func (c *OpenStack) Metric() (*gophercloud.ServiceClient, error) {
	if c.metric != nil {
		return c.metric, nil
	}
	client, err := openstack.NewMetricV1(context.Background(), c.Provider, c.EndpointOpts)
	if err != nil {
		return nil, fmt.Errorf("creating metric client: %w", err)
	}
	c.metric = client
	return c.metric, nil
}

type QuotaUsage struct {
	InUse int
	Limit int
}

type QuotaInfo struct {
	VCPUs   QuotaUsage
	RAM     QuotaUsage // in MB
	Storage QuotaUsage // in GB
	Volumes QuotaUsage
}

func (c *OpenStack) FetchQuotas(ctx context.Context) QuotaInfo {
	var info QuotaInfo

	// Compute quotas (CPUs, RAM)
	computeClient, err := c.Compute()
	if err == nil {
		detail, err := computequotas.GetDetail(ctx, computeClient, c.ProjectID).Extract()
		if err == nil {
			info.VCPUs = QuotaUsage{InUse: detail.Cores.InUse, Limit: detail.Cores.Limit}
			info.RAM = QuotaUsage{InUse: detail.RAM.InUse, Limit: detail.RAM.Limit}
		}
	}

	// Block storage quotas (Gigabytes)
	bsClient, err := c.BlockStorage()
	if err == nil {
		usage, err := bsquotas.GetUsage(ctx, bsClient, c.ProjectID).Extract()
		if err == nil {
			info.Storage = QuotaUsage{InUse: usage.Gigabytes.InUse, Limit: usage.Gigabytes.Limit}
			info.Volumes = QuotaUsage{InUse: usage.Volumes.InUse, Limit: usage.Volumes.Limit}
		}
	}

	return info
}

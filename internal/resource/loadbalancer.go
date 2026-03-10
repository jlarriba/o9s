package resource

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/loadbalancers"
	"github.com/jlarriba/o9s/internal/client"
)

func init() {
	Register(&LoadBalancer{})
	Alias("loadbalancers", "loadbalancer")
	Alias("lb", "loadbalancer")
}

type LoadBalancer struct{}

func (l *LoadBalancer) Kind() string  { return "loadbalancer" }
func (l *LoadBalancer) IDColumn() int { return 1 }

func (l *LoadBalancer) Columns() []Column {
	return []Column{
		{Name: "NAME", Width: 20},
		{Name: "ID", Width: 16},
		{Name: "PROV STATUS", Width: 14},
		{Name: "OP STATUS", Width: 10},
		{Name: "VIP ADDRESS", Width: 16},
		{Name: "PROVIDER", Width: 10},
	}
}

func (l *LoadBalancer) List(ctx context.Context, c *client.OpenStack) ([][]string, error) {
	lbClient, err := c.LoadBalancer()
	if err != nil {
		return nil, err
	}

	allPages, err := loadbalancers.List(lbClient, loadbalancers.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing load balancers: %w", err)
	}
	allLBs, err := loadbalancers.ExtractLoadBalancers(allPages)
	if err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(allLBs))
	for _, lb := range allLBs {
		rows = append(rows, []string{
			lb.Name,
			lb.ID,
			lb.ProvisioningStatus,
			lb.OperatingStatus,
			lb.VipAddress,
			lb.Provider,
		})
	}
	return rows, nil
}

func (l *LoadBalancer) Show(ctx context.Context, c *client.OpenStack, id string) ([][2]string, error) {
	lbClient, err := c.LoadBalancer()
	if err != nil {
		return nil, err
	}

	lb, err := loadbalancers.Get(ctx, lbClient, id).Extract()
	if err != nil {
		return nil, fmt.Errorf("getting load balancer %s: %w", id, err)
	}

	netNames := BuildNetworkNameMap(ctx, c)
	subnetNames := buildSubnetNameMap(ctx, c)

	return [][2]string{
		{"Name", lb.Name},
		{"ID", lb.ID},
		{"Description", lb.Description},
		{"Provisioning Status", lb.ProvisioningStatus},
		{"Operating Status", lb.OperatingStatus},
		{"Admin State Up", fmt.Sprintf("%v", lb.AdminStateUp)},
		{"VIP Address", lb.VipAddress},
		{"VIP Network", ResolveName(lb.VipNetworkID, netNames)},
		{"VIP Subnet", ResolveName(lb.VipSubnetID, subnetNames)},
		{"VIP Port ID", lb.VipPortID},
		{"Provider", lb.Provider},
		{"Flavor ID", lb.FlavorID},
		{"Availability Zone", lb.AvailabilityZone},
		{"Project ID", lb.ProjectID},
		{"Created", lb.CreatedAt.String()},
		{"Updated", lb.UpdatedAt.String()},
	}, nil
}

func (l *LoadBalancer) Delete(ctx context.Context, c *client.OpenStack, id string) error {
	lbClient, err := c.LoadBalancer()
	if err != nil {
		return err
	}
	return loadbalancers.Delete(ctx, lbClient, id, loadbalancers.DeleteOpts{}).ExtractErr()
}

package resource

import (
	"context"
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/jlarriba/o9s/internal/client"
)

func init() {
	Register(&Server{})
	Alias("srv", "server")
	Alias("servers", "server")
	Alias("instance", "server")
	Alias("vm", "server")
}

type Server struct{}

func (s *Server) Kind() string  { return "server" }
func (s *Server) IDColumn() int { return 1 }

func (s *Server) Columns() []Column {
	return []Column{
		{Name: "NAME", Width: 20},
		{Name: "ID", Width: 16},
		{Name: "STATUS", Width: 10},
		{Name: "FLAVOR", Width: 12},
		{Name: "IMAGE", Width: 20},
		{Name: "IPs", Width: 0},
	}
}

func (s *Server) List(ctx context.Context, c *client.OpenStack) ([][]string, error) {
	computeClient, err := c.Compute()
	if err != nil {
		return nil, err
	}

	allPages, err := servers.List(computeClient, servers.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing servers: %w", err)
	}
	allServers, err := servers.ExtractServers(allPages)
	if err != nil {
		return nil, err
	}

	// Build image ID→name lookup
	imageNames := buildImageNameMap(ctx, c)
	// Build flavor ID→name lookup
	flavorNames := buildFlavorNameMap(ctx, c)

	rows := make([][]string, 0, len(allServers))
	for _, srv := range allServers {
		rows = append(rows, []string{
			srv.Name,
			srv.ID,
			srv.Status,
			resolveFlavorName(srv, flavorNames),
			resolveImageName(srv, imageNames),
			extractIPs(srv),
		})
	}
	return rows, nil
}

func (s *Server) Show(ctx context.Context, c *client.OpenStack, id string) ([][2]string, error) {
	computeClient, err := c.Compute()
	if err != nil {
		return nil, err
	}

	srv, err := servers.Get(ctx, computeClient, id).Extract()
	if err != nil {
		return nil, fmt.Errorf("getting server %s: %w", id, err)
	}

	imageNames := buildImageNameMap(ctx, c)
	flavorNames := buildFlavorNameMap(ctx, c)

	return [][2]string{
		{"Name", srv.Name},
		{"ID", srv.ID},
		{"Status", srv.Status},
		{"Tenant ID", srv.TenantID},
		{"User ID", srv.UserID},
		{"Host ID", srv.HostID},
		{"Flavor", resolveFlavorName(*srv, flavorNames)},
		{"Image", resolveImageName(*srv, imageNames)},
		{"IPs", extractIPs(*srv)},
		{"Key Name", srv.KeyName},
		{"Created", srv.Created.String()},
		{"Updated", srv.Updated.String()},
		{"Availability Zone", srv.AvailabilityZone},
		{"Power State", fmt.Sprintf("%d", srv.PowerState)},
		{"Task State", srv.TaskState},
		{"VM State", srv.VmState},
		{"Metadata", fmt.Sprintf("%v", srv.Metadata)},
	}, nil
}

func buildImageNameMap(ctx context.Context, c *client.OpenStack) map[string]string {
	m := map[string]string{}
	imgClient, err := c.ImageService()
	if err != nil {
		return m
	}
	allPages, err := images.List(imgClient, images.ListOpts{}).AllPages(ctx)
	if err != nil {
		return m
	}
	allImages, err := images.ExtractImages(allPages)
	if err != nil {
		return m
	}
	for _, img := range allImages {
		m[img.ID] = img.Name
	}
	return m
}

func buildFlavorNameMap(ctx context.Context, c *client.OpenStack) map[string]string {
	m := map[string]string{}
	computeClient, err := c.Compute()
	if err != nil {
		return m
	}
	allPages, err := flavors.ListDetail(computeClient, flavors.ListOpts{}).AllPages(ctx)
	if err != nil {
		return m
	}
	allFlavors, err := flavors.ExtractFlavors(allPages)
	if err != nil {
		return m
	}
	for _, f := range allFlavors {
		m[f.ID] = f.Name
	}
	return m
}

func resolveFlavorName(srv servers.Server, names map[string]string) string {
	// Try original_name from the server response first
	if name, ok := srv.Flavor["original_name"].(string); ok {
		return name
	}
	// Fall back to lookup by ID
	if id, ok := srv.Flavor["id"].(string); ok {
		if name, ok := names[id]; ok {
			return name
		}
		return id
	}
	return ""
}

func resolveImageName(srv servers.Server, names map[string]string) string {
	if len(srv.Image) == 0 {
		return "(boot vol)"
	}
	if id, ok := srv.Image["id"].(string); ok {
		if name, ok := names[id]; ok {
			return name
		}
		return id
	}
	return ""
}

func extractIPs(srv servers.Server) string {
	var ips []string
	for _, addrs := range srv.Addresses {
		addrList, ok := addrs.([]interface{})
		if !ok {
			continue
		}
		for _, a := range addrList {
			addrMap, ok := a.(map[string]interface{})
			if !ok {
				continue
			}
			if addr, ok := addrMap["addr"].(string); ok {
				ips = append(ips, addr)
			}
		}
	}
	return strings.Join(ips, ", ")
}

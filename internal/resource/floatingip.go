package resource

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/jlarriba/o9s/internal/client"
)

func init() {
	Register(&FloatingIP{})
	Alias("floatingips", "floatingip")
	Alias("fip", "floatingip")
	Alias("fips", "floatingip")
}

type FloatingIP struct{}

func (f *FloatingIP) Kind() string  { return "floatingip" }
func (f *FloatingIP) IDColumn() int { return 1 }

func (f *FloatingIP) Columns() []Column {
	return []Column{
		{Name: "FLOATING IP", Width: 15},
		{Name: "ID", Width: 16},
		{Name: "STATUS", Width: 10},
		{Name: "FIXED IP", Width: 15},
		{Name: "ROUTER", Width: 16},
		{Name: "NETWORK", Width: 16},
	}
}

func (f *FloatingIP) List(ctx context.Context, c *client.OpenStack) ([][]string, error) {
	netClient, err := c.Network()
	if err != nil {
		return nil, err
	}

	allPages, err := floatingips.List(netClient, floatingips.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing floating IPs: %w", err)
	}
	allFIPs, err := floatingips.ExtractFloatingIPs(allPages)
	if err != nil {
		return nil, err
	}

	netNames := BuildNetworkNameMap(ctx, c)
	routerNames := buildRouterNameMap(ctx, c)

	rows := make([][]string, 0, len(allFIPs))
	for _, fip := range allFIPs {
		rows = append(rows, []string{
			fip.FloatingIP,
			fip.ID,
			fip.Status,
			fip.FixedIP,
			ResolveName(fip.RouterID, routerNames),
			ResolveName(fip.FloatingNetworkID, netNames),
		})
	}
	return rows, nil
}

func (f *FloatingIP) Show(ctx context.Context, c *client.OpenStack, id string) ([][2]string, error) {
	netClient, err := c.Network()
	if err != nil {
		return nil, err
	}

	fip, err := floatingips.Get(ctx, netClient, id).Extract()
	if err != nil {
		return nil, fmt.Errorf("getting floating IP %s: %w", id, err)
	}

	netNames := BuildNetworkNameMap(ctx, c)
	routerNames := buildRouterNameMap(ctx, c)

	return [][2]string{
		{"Floating IP", fip.FloatingIP},
		{"ID", fip.ID},
		{"Status", fip.Status},
		{"Fixed IP", fip.FixedIP},
		{"Port ID", fip.PortID},
		{"Network", ResolveName(fip.FloatingNetworkID, netNames)},
		{"Router", ResolveName(fip.RouterID, routerNames)},
		{"Tenant ID", fip.TenantID},
		{"Description", fip.Description},
		{"Created At", fip.CreatedAt.String()},
		{"Updated At", fip.UpdatedAt.String()},
	}, nil
}

func (f *FloatingIP) Delete(ctx context.Context, c *client.OpenStack, id string) error {
	netClient, err := c.Network()
	if err != nil {
		return err
	}
	return floatingips.Delete(ctx, netClient, id).ExtractErr()
}

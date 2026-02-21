package resource

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	"github.com/jlarriba/o9s/internal/client"
)

func init() {
	Register(&Router{})
	Alias("routers", "router")
	Alias("rtr", "router")
}

type Router struct{}

func (r *Router) Kind() string  { return "router" }
func (r *Router) IDColumn() int { return 1 }

func (r *Router) Columns() []Column {
	return []Column{
		{Name: "NAME", Width: 20},
		{Name: "ID", Width: 16},
		{Name: "STATUS", Width: 10},
		{Name: "ADMIN STATE", Width: 10},
		{Name: "EXT GATEWAY", Width: 16},
	}
}

func (r *Router) List(ctx context.Context, c *client.OpenStack) ([][]string, error) {
	netClient, err := c.Network()
	if err != nil {
		return nil, err
	}

	allPages, err := routers.List(netClient, routers.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing routers: %w", err)
	}
	allRouters, err := routers.ExtractRouters(allPages)
	if err != nil {
		return nil, err
	}

	netNames := BuildNetworkNameMap(ctx, c)

	rows := make([][]string, 0, len(allRouters))
	for _, rtr := range allRouters {
		rows = append(rows, []string{
			rtr.Name,
			rtr.ID,
			rtr.Status,
			fmt.Sprintf("%v", rtr.AdminStateUp),
			ResolveName(rtr.GatewayInfo.NetworkID, netNames),
		})
	}
	return rows, nil
}

func (r *Router) Show(ctx context.Context, c *client.OpenStack, id string) ([][2]string, error) {
	netClient, err := c.Network()
	if err != nil {
		return nil, err
	}

	rtr, err := routers.Get(ctx, netClient, id).Extract()
	if err != nil {
		return nil, fmt.Errorf("getting router %s: %w", id, err)
	}

	netNames := BuildNetworkNameMap(ctx, c)

	return [][2]string{
		{"Name", rtr.Name},
		{"ID", rtr.ID},
		{"Status", rtr.Status},
		{"Admin State Up", fmt.Sprintf("%v", rtr.AdminStateUp)},
		{"Tenant ID", rtr.TenantID},
		{"External Gateway", ResolveName(rtr.GatewayInfo.NetworkID, netNames)},
		{"Description", rtr.Description},
	}, nil
}

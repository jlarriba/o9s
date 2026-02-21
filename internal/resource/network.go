package resource

import (
	"context"
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/jlarriba/o9s/internal/client"
)

func init() {
	Register(&Network{})
	Alias("net", "network")
	Alias("networks", "network")
}

type Network struct{}

func (n *Network) Kind() string    { return "network" }
func (n *Network) IDColumn() int { return 1 }

func (n *Network) Columns() []Column {
	return []Column{
		{Name: "NAME", Width: 20},
		{Name: "ID", Width: 16},
		{Name: "STATUS", Width: 10},
		{Name: "SUBNETS", Width: 0},
		{Name: "SHARED", Width: 8},
		{Name: "ADMIN UP", Width: 8},
	}
}

func (n *Network) List(ctx context.Context, c *client.OpenStack) ([][]string, error) {
	netClient, err := c.Network()
	if err != nil {
		return nil, err
	}

	allPages, err := networks.List(netClient, networks.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing networks: %w", err)
	}
	allNetworks, err := networks.ExtractNetworks(allPages)
	if err != nil {
		return nil, err
	}

	subnetNames := buildSubnetNameMap(ctx, c)

	rows := make([][]string, 0, len(allNetworks))
	for _, net := range allNetworks {
		rows = append(rows, []string{
			net.Name,
			net.ID,
			net.Status,
			resolveIDs(net.Subnets, subnetNames),
			fmt.Sprintf("%v", net.Shared),
			fmt.Sprintf("%v", net.AdminStateUp),
		})
	}
	return rows, nil
}

func (n *Network) Show(ctx context.Context, c *client.OpenStack, id string) ([][2]string, error) {
	netClient, err := c.Network()
	if err != nil {
		return nil, err
	}

	net, err := networks.Get(ctx, netClient, id).Extract()
	if err != nil {
		return nil, fmt.Errorf("getting network %s: %w", id, err)
	}

	return [][2]string{
		{"Name", net.Name},
		{"ID", net.ID},
		{"Status", net.Status},
		{"Tenant ID", net.TenantID},
		{"Admin State Up", fmt.Sprintf("%v", net.AdminStateUp)},
		{"Shared", fmt.Sprintf("%v", net.Shared)},
		{"Subnets", resolveIDs(net.Subnets, buildSubnetNameMap(ctx, c))},
		{"Availability Zone Hints", strings.Join(net.AvailabilityZoneHints, ", ")},
		{"Description", net.Description},
	}, nil
}

package resource

import (
	"context"
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	"github.com/jlarriba/o9s/internal/client"
)

func init() {
	Register(&Subnet{})
	Alias("subnets", "subnet")
	Alias("sub", "subnet")
}

type Subnet struct{}

func (s *Subnet) Kind() string  { return "subnet" }
func (s *Subnet) IDColumn() int { return 1 }

func (s *Subnet) Columns() []Column {
	return []Column{
		{Name: "NAME", Width: 20},
		{Name: "ID", Width: 16},
		{Name: "CIDR", Width: 18},
		{Name: "GATEWAY", Width: 15},
		{Name: "NETWORK", Width: 16},
		{Name: "IP VERSION", Width: 10},
	}
}

func (s *Subnet) List(ctx context.Context, c *client.OpenStack) ([][]string, error) {
	netClient, err := c.Network()
	if err != nil {
		return nil, err
	}

	allPages, err := subnets.List(netClient, subnets.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing subnets: %w", err)
	}
	allSubnets, err := subnets.ExtractSubnets(allPages)
	if err != nil {
		return nil, err
	}

	netNames := BuildNetworkNameMap(ctx, c)

	rows := make([][]string, 0, len(allSubnets))
	for _, sub := range allSubnets {
		rows = append(rows, []string{
			sub.Name,
			sub.ID,
			sub.CIDR,
			sub.GatewayIP,
			ResolveName(sub.NetworkID, netNames),
			fmt.Sprintf("%d", sub.IPVersion),
		})
	}
	return rows, nil
}

func (s *Subnet) Show(ctx context.Context, c *client.OpenStack, id string) ([][2]string, error) {
	netClient, err := c.Network()
	if err != nil {
		return nil, err
	}

	sub, err := subnets.Get(ctx, netClient, id).Extract()
	if err != nil {
		return nil, fmt.Errorf("getting subnet %s: %w", id, err)
	}

	netNames := BuildNetworkNameMap(ctx, c)

	var dns []string
	dns = append(dns, sub.DNSNameservers...)

	var pools []string
	for _, p := range sub.AllocationPools {
		pools = append(pools, fmt.Sprintf("%s-%s", p.Start, p.End))
	}

	return [][2]string{
		{"Name", sub.Name},
		{"ID", sub.ID},
		{"Network", ResolveName(sub.NetworkID, netNames)},
		{"CIDR", sub.CIDR},
		{"Gateway IP", sub.GatewayIP},
		{"IP Version", fmt.Sprintf("%d", sub.IPVersion)},
		{"Enable DHCP", fmt.Sprintf("%v", sub.EnableDHCP)},
		{"DNS Nameservers", strings.Join(dns, ", ")},
		{"Allocation Pools", strings.Join(pools, ", ")},
		{"Tenant ID", sub.TenantID},
		{"Description", sub.Description},
	}, nil
}

func (s *Subnet) Delete(ctx context.Context, c *client.OpenStack, id string) error {
	netClient, err := c.Network()
	if err != nil {
		return err
	}
	return subnets.Delete(ctx, netClient, id).ExtractErr()
}

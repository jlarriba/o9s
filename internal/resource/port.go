package resource

import (
	"context"
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/jlarriba/o9s/internal/client"
)

func init() {
	Register(&Port{})
	Alias("ports", "port")
}

type Port struct{}

func (p *Port) Kind() string  { return "port" }
func (p *Port) IDColumn() int { return 1 }

func (p *Port) Columns() []Column {
	return []Column{
		{Name: "NAME", Width: 15},
		{Name: "ID", Width: 16},
		{Name: "STATUS", Width: 10},
		{Name: "MAC ADDRESS", Width: 17},
		{Name: "FIXED IPs", Width: 0},
		{Name: "NETWORK", Width: 16},
	}
}

func (p *Port) List(ctx context.Context, c *client.OpenStack) ([][]string, error) {
	netClient, err := c.Network()
	if err != nil {
		return nil, err
	}

	allPages, err := ports.List(netClient, ports.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing ports: %w", err)
	}
	allPorts, err := ports.ExtractPorts(allPages)
	if err != nil {
		return nil, err
	}

	netNames := BuildNetworkNameMap(ctx, c)

	rows := make([][]string, 0, len(allPorts))
	for _, port := range allPorts {
		var ips []string
		for _, fip := range port.FixedIPs {
			ips = append(ips, fip.IPAddress)
		}
		rows = append(rows, []string{
			port.Name,
			port.ID,
			port.Status,
			port.MACAddress,
			strings.Join(ips, ", "),
			ResolveName(port.NetworkID, netNames),
		})
	}
	return rows, nil
}

func (p *Port) Show(ctx context.Context, c *client.OpenStack, id string) ([][2]string, error) {
	netClient, err := c.Network()
	if err != nil {
		return nil, err
	}

	port, err := ports.Get(ctx, netClient, id).Extract()
	if err != nil {
		return nil, fmt.Errorf("getting port %s: %w", id, err)
	}

	netNames := BuildNetworkNameMap(ctx, c)
	serverNames := buildServerNameMap(ctx, c)

	var ips []string
	for _, fip := range port.FixedIPs {
		ips = append(ips, fmt.Sprintf("%s (subnet: %s)", fip.IPAddress, fip.SubnetID))
	}

	deviceName := port.DeviceID
	if port.DeviceOwner == "compute:nova" || strings.HasPrefix(port.DeviceOwner, "compute:") {
		deviceName = ResolveName(port.DeviceID, serverNames)
	}

	return [][2]string{
		{"Name", port.Name},
		{"ID", port.ID},
		{"Status", port.Status},
		{"MAC Address", port.MACAddress},
		{"Fixed IPs", strings.Join(ips, "; ")},
		{"Network", ResolveName(port.NetworkID, netNames)},
		{"Device Owner", port.DeviceOwner},
		{"Device", deviceName},
		{"Tenant ID", port.TenantID},
		{"Security Groups", strings.Join(port.SecurityGroups, ", ")},
		{"Admin State Up", fmt.Sprintf("%v", port.AdminStateUp)},
		{"Description", port.Description},
	}, nil
}

func (p *Port) Delete(ctx context.Context, c *client.OpenStack, id string) error {
	netClient, err := c.Network()
	if err != nil {
		return err
	}
	return ports.Delete(ctx, netClient, id).ExtractErr()
}

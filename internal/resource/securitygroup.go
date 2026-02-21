package resource

import (
	"context"
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"github.com/jlarriba/o9s/internal/client"
)

func init() {
	Register(&SecurityGroup{})
	Alias("securitygroups", "securitygroup")
	Alias("sg", "securitygroup")
	Alias("secgroup", "securitygroup")
}

type SecurityGroup struct{}

func (s *SecurityGroup) Kind() string    { return "securitygroup" }
func (s *SecurityGroup) IDColumn() int { return 1 }

func (s *SecurityGroup) Columns() []Column {
	return []Column{
		{Name: "NAME", Width: 20},
		{Name: "ID", Width: 16},
		{Name: "DESCRIPTION", Width: 0},
	}
}

func (s *SecurityGroup) List(ctx context.Context, c *client.OpenStack) ([][]string, error) {
	netClient, err := c.Network()
	if err != nil {
		return nil, err
	}

	allPages, err := groups.List(netClient, groups.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing security groups: %w", err)
	}
	allGroups, err := groups.ExtractGroups(allPages)
	if err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(allGroups))
	for _, sg := range allGroups {
		rows = append(rows, []string{
			sg.Name,
			sg.ID,
			sg.Description,
		})
	}
	return rows, nil
}

func (s *SecurityGroup) Show(ctx context.Context, c *client.OpenStack, id string) ([][2]string, error) {
	netClient, err := c.Network()
	if err != nil {
		return nil, err
	}

	sg, err := groups.Get(ctx, netClient, id).Extract()
	if err != nil {
		return nil, fmt.Errorf("getting security group %s: %w", id, err)
	}

	var rules []string
	for _, r := range sg.Rules {
		portRange := ""
		if r.PortRangeMin > 0 || r.PortRangeMax > 0 {
			portRange = fmt.Sprintf("%d-%d", r.PortRangeMin, r.PortRangeMax)
		}
		proto := r.Protocol
		if proto == "" {
			proto = "any"
		}
		remote := r.RemoteIPPrefix
		if remote == "" {
			remote = "any"
		}
		rules = append(rules, fmt.Sprintf("%s %s %s %s %s",
			r.Direction, r.EtherType, proto, portRange, remote))
	}

	return [][2]string{
		{"Name", sg.Name},
		{"ID", sg.ID},
		{"Description", sg.Description},
		{"Tenant ID", sg.TenantID},
		{"Rules", strings.Join(rules, "\n")},
	}, nil
}

func (s *SecurityGroup) Delete(ctx context.Context, c *client.OpenStack, id string) error {
	netClient, err := c.Network()
	if err != nil {
		return err
	}
	return groups.Delete(ctx, netClient, id).ExtractErr()
}

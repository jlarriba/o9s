package resource

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/dns/v2/zones"
	"github.com/jlarriba/o9s/internal/client"
)

func init() {
	Register(&Zone{})
	Alias("zones", "zone")
	Alias("dns", "zone")
}

type Zone struct{}

func (z *Zone) Kind() string  { return "zone" }
func (z *Zone) IDColumn() int { return 1 }

func (z *Zone) Columns() []Column {
	return []Column{
		{Name: "NAME", Width: 30},
		{Name: "ID", Width: 16},
		{Name: "STATUS", Width: 10},
		{Name: "TYPE", Width: 10},
		{Name: "TTL", Width: 8},
		{Name: "DESCRIPTION", Width: 0},
	}
}

func (z *Zone) List(ctx context.Context, c *client.OpenStack) ([][]string, error) {
	dnsClient, err := c.DNS()
	if err != nil {
		return nil, err
	}

	allPages, err := zones.List(dnsClient, zones.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing zones: %w", err)
	}
	allZones, err := zones.ExtractZones(allPages)
	if err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(allZones))
	for _, zone := range allZones {
		rows = append(rows, []string{
			zone.Name,
			zone.ID,
			zone.Status,
			zone.Type,
			fmt.Sprintf("%d", zone.TTL),
			zone.Description,
		})
	}
	return rows, nil
}

func (z *Zone) Show(ctx context.Context, c *client.OpenStack, id string) ([][2]string, error) {
	dnsClient, err := c.DNS()
	if err != nil {
		return nil, err
	}

	zone, err := zones.Get(ctx, dnsClient, id).Extract()
	if err != nil {
		return nil, fmt.Errorf("getting zone %s: %w", id, err)
	}

	return [][2]string{
		{"Name", zone.Name},
		{"ID", zone.ID},
		{"Status", zone.Status},
		{"Type", zone.Type},
		{"Email", zone.Email},
		{"Description", zone.Description},
		{"TTL", fmt.Sprintf("%d", zone.TTL)},
		{"Serial", fmt.Sprintf("%d", zone.Serial)},
		{"Version", fmt.Sprintf("%d", zone.Version)},
		{"Project ID", zone.ProjectID},
		{"Pool ID", zone.PoolID},
		{"Action", zone.Action},
		{"Created", zone.CreatedAt.String()},
		{"Updated", zone.UpdatedAt.String()},
	}, nil
}

func (z *Zone) Delete(ctx context.Context, c *client.OpenStack, id string) error {
	dnsClient, err := c.DNS()
	if err != nil {
		return err
	}
	_, err = zones.Delete(ctx, dnsClient, id).Extract()
	return err
}

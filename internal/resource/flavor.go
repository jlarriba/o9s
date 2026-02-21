package resource

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/jlarriba/o9s/internal/client"
)

func init() {
	Register(&Flavor{})
	Alias("flavors", "flavor")
	Alias("flv", "flavor")
}

type Flavor struct{}

func (f *Flavor) Kind() string    { return "flavor" }
func (f *Flavor) IDColumn() int { return 1 }

func (f *Flavor) Columns() []Column {
	return []Column{
		{Name: "NAME", Width: 20},
		{Name: "ID", Width: 16},
		{Name: "VCPUS", Width: 6},
		{Name: "RAM", Width: 8},
		{Name: "DISK", Width: 8},
		{Name: "PUBLIC", Width: 7},
	}
}

func (f *Flavor) List(ctx context.Context, c *client.OpenStack) ([][]string, error) {
	computeClient, err := c.Compute()
	if err != nil {
		return nil, err
	}

	allPages, err := flavors.ListDetail(computeClient, flavors.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing flavors: %w", err)
	}
	allFlavors, err := flavors.ExtractFlavors(allPages)
	if err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(allFlavors))
	for _, flv := range allFlavors {
		rows = append(rows, []string{
			flv.Name,
			flv.ID,
			fmt.Sprintf("%d", flv.VCPUs),
			fmt.Sprintf("%d MB", flv.RAM),
			fmt.Sprintf("%d GB", flv.Disk),
			fmt.Sprintf("%v", flv.IsPublic),
		})
	}
	return rows, nil
}

func (f *Flavor) Show(ctx context.Context, c *client.OpenStack, id string) ([][2]string, error) {
	computeClient, err := c.Compute()
	if err != nil {
		return nil, err
	}

	flv, err := flavors.Get(ctx, computeClient, id).Extract()
	if err != nil {
		return nil, fmt.Errorf("getting flavor %s: %w", id, err)
	}

	return [][2]string{
		{"Name", flv.Name},
		{"ID", flv.ID},
		{"VCPUs", fmt.Sprintf("%d", flv.VCPUs)},
		{"RAM", fmt.Sprintf("%d MB", flv.RAM)},
		{"Disk", fmt.Sprintf("%d GB", flv.Disk)},
		{"Ephemeral", fmt.Sprintf("%d GB", flv.Ephemeral)},
		{"Swap", fmt.Sprintf("%d MB", flv.Swap)},
		{"RxTx Factor", fmt.Sprintf("%.1f", flv.RxTxFactor)},
		{"Is Public", fmt.Sprintf("%v", flv.IsPublic)},
		{"Description", flv.Description},
	}, nil
}

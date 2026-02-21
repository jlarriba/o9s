package resource

import (
	"context"
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/jlarriba/o9s/internal/client"
)

func init() {
	Register(&Volume{})
	Alias("vol", "volume")
	Alias("volumes", "volume")
}

type Volume struct{}

func (v *Volume) Kind() string  { return "volume" }
func (v *Volume) IDColumn() int { return 1 }

func (v *Volume) Columns() []Column {
	return []Column{
		{Name: "NAME", Width: 20},
		{Name: "ID", Width: 16},
		{Name: "STATUS", Width: 10},
		{Name: "SIZE", Width: 6},
		{Name: "TYPE", Width: 12},
		{Name: "ATTACHED TO", Width: 0},
	}
}

func (v *Volume) List(ctx context.Context, c *client.OpenStack) ([][]string, error) {
	bsClient, err := c.BlockStorage()
	if err != nil {
		return nil, err
	}

	allPages, err := volumes.List(bsClient, volumes.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing volumes: %w", err)
	}
	allVolumes, err := volumes.ExtractVolumes(allPages)
	if err != nil {
		return nil, err
	}

	serverNames := buildServerNameMap(ctx, c)

	rows := make([][]string, 0, len(allVolumes))
	for _, vol := range allVolumes {
		var attached []string
		for _, att := range vol.Attachments {
			attached = append(attached, ResolveName(att.ServerID, serverNames))
		}
		rows = append(rows, []string{
			vol.Name,
			vol.ID,
			vol.Status,
			fmt.Sprintf("%d GB", vol.Size),
			vol.VolumeType,
			strings.Join(attached, ", "),
		})
	}
	return rows, nil
}

func (v *Volume) Show(ctx context.Context, c *client.OpenStack, id string) ([][2]string, error) {
	bsClient, err := c.BlockStorage()
	if err != nil {
		return nil, err
	}

	vol, err := volumes.Get(ctx, bsClient, id).Extract()
	if err != nil {
		return nil, fmt.Errorf("getting volume %s: %w", id, err)
	}

	serverNames := buildServerNameMap(ctx, c)

	var attachments []string
	for _, att := range vol.Attachments {
		attachments = append(attachments, fmt.Sprintf("%s → %s", ResolveName(att.ServerID, serverNames), att.Device))
	}

	return [][2]string{
		{"Name", vol.Name},
		{"ID", vol.ID},
		{"Status", vol.Status},
		{"Size", fmt.Sprintf("%d GB", vol.Size)},
		{"Volume Type", vol.VolumeType},
		{"Description", vol.Description},
		{"Availability Zone", vol.AvailabilityZone},
		{"Created At", vol.CreatedAt.String()},
		{"Attachments", strings.Join(attachments, "; ")},
		{"Bootable", vol.Bootable},
		{"Encrypted", fmt.Sprintf("%v", vol.Encrypted)},
		{"Multiattach", fmt.Sprintf("%v", vol.Multiattach)},
		{"Snapshot ID", vol.SnapshotID},
		{"Source Vol ID", vol.SourceVolID},
		{"Metadata", fmt.Sprintf("%v", vol.Metadata)},
	}, nil
}

func (v *Volume) Delete(ctx context.Context, c *client.OpenStack, id string) error {
	bsClient, err := c.BlockStorage()
	if err != nil {
		return err
	}
	return volumes.Delete(ctx, bsClient, id, volumes.DeleteOpts{}).ExtractErr()
}

package resource

import (
	"context"
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/jlarriba/o9s/internal/client"
)

func init() {
	Register(&Image{})
	Alias("images", "image")
	Alias("img", "image")
}

type Image struct{}

func (i *Image) Kind() string    { return "image" }
func (i *Image) IDColumn() int { return 1 }

func (i *Image) Columns() []Column {
	return []Column{
		{Name: "NAME", Width: 25},
		{Name: "ID", Width: 16},
		{Name: "STATUS", Width: 10},
		{Name: "SIZE", Width: 10},
		{Name: "VISIBILITY", Width: 10},
		{Name: "MIN DISK", Width: 8},
	}
}

func (i *Image) List(ctx context.Context, c *client.OpenStack) ([][]string, error) {
	imgClient, err := c.ImageService()
	if err != nil {
		return nil, err
	}

	allPages, err := images.List(imgClient, images.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing images: %w", err)
	}
	allImages, err := images.ExtractImages(allPages)
	if err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(allImages))
	for _, img := range allImages {
		rows = append(rows, []string{
			img.Name,
			img.ID,
			string(img.Status),
			formatBytes(img.SizeBytes),
			string(img.Visibility),
			fmt.Sprintf("%d GB", img.MinDiskGigabytes),
		})
	}
	return rows, nil
}

func (i *Image) Show(ctx context.Context, c *client.OpenStack, id string) ([][2]string, error) {
	imgClient, err := c.ImageService()
	if err != nil {
		return nil, err
	}

	img, err := images.Get(ctx, imgClient, id).Extract()
	if err != nil {
		return nil, fmt.Errorf("getting image %s: %w", id, err)
	}

	return [][2]string{
		{"Name", img.Name},
		{"ID", img.ID},
		{"Status", string(img.Status)},
		{"Size", formatBytes(img.SizeBytes)},
		{"Min Disk", fmt.Sprintf("%d GB", img.MinDiskGigabytes)},
		{"Min RAM", fmt.Sprintf("%d MB", img.MinRAMMegabytes)},
		{"Visibility", string(img.Visibility)},
		{"Container Format", img.ContainerFormat},
		{"Disk Format", img.DiskFormat},
		{"Created At", img.CreatedAt.String()},
		{"Updated At", img.UpdatedAt.String()},
		{"Owner", img.Owner},
		{"Checksum", img.Checksum},
		{"Tags", strings.Join(img.Tags, ", ")},
	}, nil
}

func (i *Image) Delete(ctx context.Context, c *client.OpenStack, id string) error {
	imgClient, err := c.ImageService()
	if err != nil {
		return err
	}
	return images.Delete(ctx, imgClient, id).ExtractErr()
}

func formatBytes(b int64) string {
	if b == 0 {
		return "0"
	}
	const gb = 1024 * 1024 * 1024
	const mb = 1024 * 1024
	if b >= gb {
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	}
	return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
}

package resource

import (
	"context"
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
	"github.com/jlarriba/o9s/internal/client"
)

func init() {
	Register(&Project{})
	Alias("projects", "project")
	Alias("proj", "project")
	Alias("tenant", "project")
}

type Project struct{}

func (p *Project) Kind() string    { return "project" }
func (p *Project) IDColumn() int { return 1 }

func (p *Project) Columns() []Column {
	return []Column{
		{Name: "NAME", Width: 25},
		{Name: "ID", Width: 16},
		{Name: "ENABLED", Width: 8},
		{Name: "DOMAIN ID", Width: 16},
		{Name: "DESCRIPTION", Width: 0},
	}
}

func (p *Project) List(ctx context.Context, c *client.OpenStack) ([][]string, error) {
	idClient, err := c.Identity()
	if err != nil {
		return nil, err
	}

	allPages, err := projects.List(idClient, projects.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}
	allProjects, err := projects.ExtractProjects(allPages)
	if err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(allProjects))
	for _, proj := range allProjects {
		rows = append(rows, []string{
			proj.Name,
			proj.ID,
			fmt.Sprintf("%v", proj.Enabled),
			proj.DomainID,
			proj.Description,
		})
	}
	return rows, nil
}

func (p *Project) Show(ctx context.Context, c *client.OpenStack, id string) ([][2]string, error) {
	idClient, err := c.Identity()
	if err != nil {
		return nil, err
	}

	proj, err := projects.Get(ctx, idClient, id).Extract()
	if err != nil {
		return nil, fmt.Errorf("getting project %s: %w", id, err)
	}

	return [][2]string{
		{"Name", proj.Name},
		{"ID", proj.ID},
		{"Domain ID", proj.DomainID},
		{"Description", proj.Description},
		{"Enabled", fmt.Sprintf("%v", proj.Enabled)},
		{"Is Domain", fmt.Sprintf("%v", proj.IsDomain)},
		{"Parent ID", proj.ParentID},
		{"Tags", strings.Join(proj.Tags, ", ")},
	}, nil
}

func (p *Project) Delete(ctx context.Context, c *client.OpenStack, id string) error {
	idClient, err := c.Identity()
	if err != nil {
		return err
	}
	return projects.Delete(ctx, idClient, id).ExtractErr()
}

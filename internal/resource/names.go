package resource

import (
	"context"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	"github.com/jlarriba/o9s/internal/client"
)

// BuildNetworkNameMap returns a map of network ID to name.
func BuildNetworkNameMap(ctx context.Context, c *client.OpenStack) map[string]string {
	m := map[string]string{}
	netClient, err := c.Network()
	if err != nil {
		return m
	}
	allPages, err := networks.List(netClient, networks.ListOpts{}).AllPages(ctx)
	if err != nil {
		return m
	}
	allNets, err := networks.ExtractNetworks(allPages)
	if err != nil {
		return m
	}
	for _, n := range allNets {
		m[n.ID] = n.Name
	}
	return m
}

func buildRouterNameMap(ctx context.Context, c *client.OpenStack) map[string]string {
	m := map[string]string{}
	netClient, err := c.Network()
	if err != nil {
		return m
	}
	allPages, err := routers.List(netClient, routers.ListOpts{}).AllPages(ctx)
	if err != nil {
		return m
	}
	allRouters, err := routers.ExtractRouters(allPages)
	if err != nil {
		return m
	}
	for _, r := range allRouters {
		m[r.ID] = r.Name
	}
	return m
}

func buildServerNameMap(ctx context.Context, c *client.OpenStack) map[string]string {
	m := map[string]string{}
	computeClient, err := c.Compute()
	if err != nil {
		return m
	}
	allPages, err := servers.List(computeClient, servers.ListOpts{}).AllPages(ctx)
	if err != nil {
		return m
	}
	allServers, err := servers.ExtractServers(allPages)
	if err != nil {
		return m
	}
	for _, s := range allServers {
		m[s.ID] = s.Name
	}
	return m
}

func buildSubnetNameMap(ctx context.Context, c *client.OpenStack) map[string]string {
	m := map[string]string{}
	netClient, err := c.Network()
	if err != nil {
		return m
	}
	allPages, err := subnets.List(netClient, subnets.ListOpts{}).AllPages(ctx)
	if err != nil {
		return m
	}
	allSubnets, err := subnets.ExtractSubnets(allPages)
	if err != nil {
		return m
	}
	for _, s := range allSubnets {
		m[s.ID] = s.Name
	}
	return m
}

// resolveIDs resolves a slice of IDs to names, joining with ", ".
func resolveIDs(ids []string, names map[string]string) string {
	resolved := make([]string, 0, len(ids))
	for _, id := range ids {
		resolved = append(resolved, ResolveName(id, names))
	}
	return strings.Join(resolved, ", ")
}

// ResolveName looks up an ID in the map, returning the name or the original ID.
func ResolveName(id string, names map[string]string) string {
	if id == "" {
		return ""
	}
	if name, ok := names[id]; ok && name != "" {
		return name
	}
	return id
}

package resource

import (
	"context"
	"fmt"
	"strings"

	"github.com/jlarriba/o9s/internal/client"
)

type Column struct {
	Name  string
	Width int // 0 = auto-expand
}

type Resource interface {
	Kind() string
	Columns() []Column
	IDColumn() int // which column index is the identifier for Show()
	List(ctx context.Context, c *client.OpenStack) ([][]string, error)
	Show(ctx context.Context, c *client.OpenStack, id string) ([][2]string, error)
	Delete(ctx context.Context, c *client.OpenStack, id string) error
}

// RelatedResource describes a navigable link from one resource to another.
type RelatedResource struct {
	Kind        string // resource kind for Resolve(), e.g. "volume"
	ID          string // the ID to pass to Show()
	DisplayName string // human-readable label for the UI
}

// Relatable is an optional interface. Resources that expose navigable
// related items implement this in addition to Resource.
type Relatable interface {
	Related(ctx context.Context, c *client.OpenStack, id string) ([]RelatedResource, error)
}

var (
	registry = map[string]Resource{}
	aliases  = map[string]string{}
)

func Register(r Resource) {
	registry[r.Kind()] = r
}

func Alias(alias, kind string) {
	aliases[alias] = kind
}

func Resolve(name string) (Resource, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if target, ok := aliases[name]; ok {
		name = target
	}
	if r, ok := registry[name]; ok {
		return r, nil
	}
	return nil, fmt.Errorf("unknown resource %q", name)
}

func AllKinds() []string {
	kinds := make([]string, 0, len(registry))
	for k := range registry {
		kinds = append(kinds, k)
	}
	return kinds
}

func AllNames() []string {
	names := make([]string, 0, len(registry)+len(aliases))
	for k := range registry {
		names = append(names, k)
	}
	for a := range aliases {
		names = append(names, a)
	}
	return names
}

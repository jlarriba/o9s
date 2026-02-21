package resource

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs"
	"github.com/jlarriba/o9s/internal/client"
)

func init() {
	Register(&Keypair{})
	Alias("keypairs", "keypair")
	Alias("key", "keypair")
	Alias("kp", "keypair")
}

type Keypair struct{}

func (k *Keypair) Kind() string    { return "keypair" }
func (k *Keypair) IDColumn() int { return 0 } // keypairs use Name as identifier

func (k *Keypair) Columns() []Column {
	return []Column{
		{Name: "NAME", Width: 25},
		{Name: "TYPE", Width: 8},
		{Name: "FINGERPRINT", Width: 0},
	}
}

func (k *Keypair) List(ctx context.Context, c *client.OpenStack) ([][]string, error) {
	computeClient, err := c.Compute()
	if err != nil {
		return nil, err
	}

	allPages, err := keypairs.List(computeClient, keypairs.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing keypairs: %w", err)
	}
	allKeypairs, err := keypairs.ExtractKeyPairs(allPages)
	if err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(allKeypairs))
	for _, kp := range allKeypairs {
		rows = append(rows, []string{
			kp.Name,
			kp.Type,
			kp.Fingerprint,
		})
	}
	return rows, nil
}

func (k *Keypair) Show(ctx context.Context, c *client.OpenStack, id string) ([][2]string, error) {
	computeClient, err := c.Compute()
	if err != nil {
		return nil, err
	}

	kp, err := keypairs.Get(ctx, computeClient, id, keypairs.GetOpts{}).Extract()
	if err != nil {
		return nil, fmt.Errorf("getting keypair %s: %w", id, err)
	}

	return [][2]string{
		{"Name", kp.Name},
		{"Type", kp.Type},
		{"Fingerprint", kp.Fingerprint},
		{"Public Key", kp.PublicKey},
		{"User ID", kp.UserID},
	}, nil
}

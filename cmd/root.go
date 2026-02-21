package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/jlarriba/o9s/internal/client"
	_ "github.com/jlarriba/o9s/internal/resource" // register all resources
	"github.com/jlarriba/o9s/internal/ui"
	"github.com/spf13/cobra"
)

var cloudName string

var rootCmd = &cobra.Command{
	Use:   "o9s",
	Short: "O9S — OpenStack TUI",
	Long:  "A terminal UI for OpenStack, inspired by k9s.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		c, err := client.New(ctx, cloudName)
		if err != nil {
			return fmt.Errorf("auth failed: %w", err)
		}

		app := ui.NewApp(c)
		return app.Run()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cloudName, "cloud", os.Getenv("OS_CLOUD"), "OpenStack cloud name from clouds.yaml")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	appconfig "github.com/asdf57/homelabc/internal/config"
	"github.com/spf13/cobra"
)

const runInventoryPublicationGroupKey = "general.inventory_publication_group"

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a homelab container for a particular inventory capture group",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if err := cfg.ValidateRun(); err != nil {
			return err
		}
		return RunNormalMode(cmd.Context(), cfg)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	flags := runCmd.Flags()
	flags.String("inventory-publication-group", "servers-inventory", "inventory capture group")

	bindRunFlag(runInventoryPublicationGroupKey, "inventory-publication-group")
}

func bindRunFlag(key, flagName string) {
	cobra.CheckErr(settings.BindPFlag(key, runCmd.Flags().Lookup(flagName)))
}

func RunNormalMode(ctx context.Context, cfg appconfig.Config) error {
	args := runDockerArgs(cfg)
	cmd := exec.CommandContext(ctx, "docker", args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run normal container: %w", err)
	}

	return nil
}

func runDockerArgs(cfg appconfig.Config) []string {
	args := []string{
		"run",
		"--rm",
		"-it",
	}
	args = append(args,
		"-w", "/homelab",
		"-e", fmt.Sprintf("INVENTORY_PUBLICATION_GROUP=%s", cfg.General.InventoryPublicationGroup),
		"-e", fmt.Sprintf("CONTAINER_MODE=%s", "normal"),
		cfg.General.Image,
	)
	return args
}

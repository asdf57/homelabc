package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	appconfig "github.com/asdf57/homelabc/internal/config"
	"github.com/spf13/cobra"
)

const runInventoryCaptureGroupKey = "general.inventory_capture_group"

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
	flags.String("inventory-capture-group", "servers", "inventory capture group")

	bindRunFlag(runInventoryCaptureGroupKey, "inventory-capture-group")
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
		"--network", "host",
	}
	args = append(args,
		"-w", "/homelab",
		"-e", fmt.Sprintf("INVENTORY_CAPTURE_GROUP=%s", cfg.General.InventoryCaptureGroup),
		"-e", fmt.Sprintf("GIT_ANSIBLE_ROLES_REPO=%s", cfg.General.AnsibleRolesRepo),
		"-e", fmt.Sprintf("GIT_ANSIBLE_ROLES_REF=%s", cfg.General.AnsibleRolesRef),
	)
	if cfg.General.StigmergyApiUrl != "" {
		args = append(args, "-e", fmt.Sprintf("STIGMERGY_API_URL=%s", cfg.General.StigmergyApiUrl))
	}
	args = append(args,
		"-e", fmt.Sprintf("CONTAINER_MODE=%s", "normal"),
		cfg.General.Image,
	)
	return args
}

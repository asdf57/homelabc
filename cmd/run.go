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
	for _, entry := range []struct{ name, key, description string }{
		{"api-token-file", "general.api_token_file", "file containing the runner's API bearer token"},
		{"ssh-private-key-file", "general.ssh_private_key_file", "runner's SSH private key file"},
		{"ssh-certificate-file", "general.ssh_certificate_file", "runner's signed SSH user certificate"},
		{"ssh-known-hosts-file", "general.ssh_known_hosts_file", "trusted SSH host keys for managed servers"},
	} {
		flags.String(entry.name, "", entry.description)
		bindRunFlag(entry.key, entry.name)
	}

	bindRunFlag(runInventoryCaptureGroupKey, "inventory-capture-group")
}

func bindRunFlag(key, flagName string) {
	cobra.CheckErr(settings.BindPFlag(key, runCmd.Flags().Lookup(flagName)))
}

func RunNormalMode(ctx context.Context, cfg appconfig.Config) error {
	for _, path := range []string{cfg.General.APITokenFile, cfg.General.SSHPrivateKeyFile, cfg.General.SSHCertificateFile, cfg.General.SSHKnownHostsFile} {
		if path == "" {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("read runner credential file: %w", err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("runner credential must be a regular file")
		}
	}
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
	for _, entry := range []struct{ source, target, environment string }{
		{cfg.General.APITokenFile, "/run/homelab/api-token", "STIGMERGY_API_TOKEN_FILE"},
		{cfg.General.SSHPrivateKeyFile, "/run/homelab/ssh-key", "ANSIBLE_PRIVATE_KEY_FILE"},
		{cfg.General.SSHCertificateFile, "/run/homelab/ssh-certificate", "ANSIBLE_CERTIFICATE_FILE"},
		{cfg.General.SSHKnownHostsFile, "/run/homelab/known-hosts", "ANSIBLE_KNOWN_HOSTS_FILE"},
	} {
		if entry.source != "" {
			args = append(args, "--mount", fmt.Sprintf("type=bind,src=%s,dst=%s,readonly", entry.source, entry.target), "-e", entry.environment+"="+entry.target)
		}
	}
	args = append(args,
		"-e", fmt.Sprintf("CONTAINER_MODE=%s", "normal"),
		cfg.General.Image,
	)
	return args
}

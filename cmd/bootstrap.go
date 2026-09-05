package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"

	appconfig "github.com/asdf57/homelabc/internal/config"
	"github.com/spf13/cobra"
)

const (
	bootstrapContainerEnvFileKey          = "bootstrap.container_env_file"
	bootstrapMountPathKey                 = "bootstrap.container_mount_path"
	bootstrapDockerSocketPathKey          = "bootstrap.docker_socket_path"
	bootstrapHostDataPathKey              = "bootstrap.host_data_path"
	bootstrapInventoryPublicationGroupKey = "bootstrap.inventory_publication_group"
	bootstrapImageKey                     = "general.image"
	gitPrivateKeyKey                      = "git.priv_key_path"
	gitPrivateKeyMountKey                 = "git.container_priv_key_path"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Bootstrap the homelab environment",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if err := cfg.ValidateBootstrap(); err != nil {
			return err
		}
		return runBootstrap(cmd.Context(), cfg)
	},
}

func init() {
	rootCmd.AddCommand(bootstrapCmd)

	flags := bootstrapCmd.Flags()
	flags.String("container-env-file", ".env", "environment file passed to the bootstrap container")
	flags.String("mount-path", "", "container path for the mounted host data (defaults to host-data-path)")
	flags.String("docker-socket-path", "/var/run/docker.sock", "path to the host Docker socket")
	flags.String("host-data-path", "", "host directory containing all persistent homelab data")
	flags.String("image", "prov", "homelab image")
	flags.String("github-private-key", "", "host private key used to clone GitHub repositories")
	flags.String("github-private-key-mount", "", "container path for the GitHub private key")

	bindBootstrapFlag(bootstrapContainerEnvFileKey, "container-env-file")
	bindBootstrapFlag(bootstrapMountPathKey, "mount-path")
	bindBootstrapFlag(bootstrapDockerSocketPathKey, "docker-socket-path")
	bindBootstrapFlag(bootstrapHostDataPathKey, "host-data-path")
	bindBootstrapFlag(bootstrapInventoryPublicationGroupKey, "inventory-publication-group")
	bindBootstrapFlag(bootstrapImageKey, "image")
	bindBootstrapFlag(gitPrivateKeyKey, "github-private-key")
	bindBootstrapFlag(gitPrivateKeyMountKey, "github-private-key-mount")
}

func bindBootstrapFlag(key, flagName string) {
	cobra.CheckErr(settings.BindPFlag(key, bootstrapCmd.Flags().Lookup(flagName)))
}

func resolveDockerGroup() (int, error) {
	group, err := user.LookupGroup("docker")
	if err != nil {
		return 0, fmt.Errorf("lookup docker group: %w", err)
	}
	return strconv.Atoi(group.Gid)
}

func resolveHomelabGroup() (int, error) {
	group, err := user.LookupGroup("homelab")
	if err != nil {
		return 0, fmt.Errorf("lookup homelab group: %w", err)
	}
	return strconv.Atoi(group.Gid)
}

func runBootstrap(ctx context.Context, cfg appconfig.Config) error {
	dockerGroup, err := resolveDockerGroup()
	if err != nil {
		return fmt.Errorf("resolve Docker group: %w", err)
	}

	homelabGroup, err := resolveHomelabGroup()
	if err != nil {
		return fmt.Errorf("resolve homelab group: %w", err)
	}

	args := bootstrapDockerArgs(cfg, dockerGroup, homelabGroup)
	cmd := exec.CommandContext(ctx, "docker", args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run bootstrap container: %w", err)
	}

	return nil
}

func bootstrapDockerArgs(cfg appconfig.Config, dockerGroup, homelabGroup int) []string {
	b := cfg.Bootstrap
	return []string{
		"run",
		"--rm",
		"-it",
		"--privileged",
		"--network", "host",
		"--group-add", fmt.Sprintf("%d", dockerGroup),
		"--group-add", fmt.Sprintf("%d", homelabGroup),
		"-v", fmt.Sprintf("%s:%s", b.DockerSocket, b.DockerSocket),
		"-w", "/homelab",
		"-e", fmt.Sprintf("HOST_DATA_PATH=%s", b.HostDataPath),
		"--env-file", b.EnvFile,
		"-v", fmt.Sprintf("%s:%s", b.HostDataPath, b.MountPath),
		"-v", fmt.Sprintf("%s:%s:ro", cfg.Git.PrivateKey, cfg.Git.PrivateKeyMount),
		"-e", fmt.Sprintf("MOUNT_GIT_SSH_KEY_PATH=%s", cfg.Git.PrivateKeyMount),
		cfg.General.Image,
	}
}

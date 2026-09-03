package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/asdf57/homelabc/schemas"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	bootstrapContainerEnvFileKey = "bootstrap.container_env_file"
	bootstrapMountPathKey        = "bootstrap.container_mount_path"
	bootstrapDockerSocketPathKey = "bootstrap.docker_socket_path"
	bootstrapHostDataPathKey     = "bootstrap.host_data_path"
	bootstrapImageKey            = "general.image"
	gitPrivKeyPathKey            = "git.priv_key_path"
	gitPrivKeyMountPathKey       = "git.container_priv_key_path"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Bootstrap the homelab environment",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config
		if cfg.Bootstrap.ContainerMetadataMountPath == "" {
			cfg.Bootstrap.ContainerMetadataMountPath = cfg.Bootstrap.ContainerMetadataHostPath
		}
		if err := validateBootstrapConfig(cfg); err != nil {
			return err
		}
		return runBootstrap(cfg)
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

	bindBootstrapFlag(bootstrapContainerEnvFileKey, "container-env-file")
	bindBootstrapFlag(bootstrapMountPathKey, "mount-path")
	bindBootstrapFlag(bootstrapDockerSocketPathKey, "docker-socket-path")
	bindBootstrapFlag(bootstrapHostDataPathKey, "host-data-path")
	bindBootstrapFlag(bootstrapImageKey, "image")
}

func bindBootstrapFlag(key, flagName string) {
	cobra.CheckErr(viper.BindPFlag(key, bootstrapCmd.Flags().Lookup(flagName)))
}

func validateBootstrapConfig(cfg schemas.Config) error {
	b := cfg.Bootstrap
	if b.ContainerEnvFilePath == "" {
		return fmt.Errorf("container env file cannot be empty")
	}
	if info, err := os.Stat(b.ContainerEnvFilePath); err != nil {
		return fmt.Errorf("access container env file %q: %w", b.ContainerEnvFilePath, err)
	} else if info.IsDir() {
		return fmt.Errorf("container env file %q is a directory", b.ContainerEnvFilePath)
	}

	if b.ContainerMetadataHostPath == "" {
		return fmt.Errorf("host data path is required")
	}
	if !filepath.IsAbs(b.ContainerMetadataHostPath) {
		return fmt.Errorf("host data path must be absolute: %q", b.ContainerMetadataHostPath)
	}
	if info, err := os.Stat(b.ContainerMetadataHostPath); err != nil {
		return fmt.Errorf("access host data path %q: %w", b.ContainerMetadataHostPath, err)
	} else if !info.IsDir() {
		return fmt.Errorf("host data path %q is not a directory", b.ContainerMetadataHostPath)
	}

	if !filepath.IsAbs(b.ContainerMetadataMountPath) {
		return fmt.Errorf("container mount path must be absolute: %q", b.ContainerMetadataMountPath)
	}

	if b.DockerSocketHostPath == "" {
		return fmt.Errorf("Docker socket path cannot be empty")
	}
	if info, err := os.Stat(b.DockerSocketHostPath); err != nil {
		return fmt.Errorf("access Docker socket %q: %w", b.DockerSocketHostPath, err)
	} else if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("%q is not a Unix socket", b.DockerSocketHostPath)
	}

	if cfg.General.Image == "" {
		return fmt.Errorf("bootstrap image cannot be empty")
	}
	if !filepath.IsAbs(cfg.Git.PrivKeyMountPath) {
		return fmt.Errorf("Git private key mount path must be absolute: %q", cfg.Git.PrivKeyMountPath)
	}

	if cfg.Git.PrivKeyHostPath == "" {
		return fmt.Errorf("Git private key path is required")
	}
	if !filepath.IsAbs(cfg.Git.PrivKeyHostPath) {
		return fmt.Errorf("Git private key path must be absolute: %q", cfg.Git.PrivKeyHostPath)
	}
	if info, err := os.Stat(cfg.Git.PrivKeyHostPath); err != nil {
		return fmt.Errorf("access Git private key %q: %w", cfg.Git.PrivKeyHostPath, err)
	} else if !info.Mode().IsRegular() {
		return fmt.Errorf("Git private key %q is not a regular file", cfg.Git.PrivKeyHostPath)
	}

	return nil
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

func runBootstrap(cfg schemas.Config) error {
	ctx := context.Background()

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

func bootstrapDockerArgs(cfg schemas.Config, dockerGroup, homelabGroup int) []string {
	b := cfg.Bootstrap
	return []string{
		"run",
		"--rm",
		"-it",
		"--privileged",
		"--network", "host",
		"--group-add", fmt.Sprintf("%d", dockerGroup),
		"--group-add", fmt.Sprintf("%d", homelabGroup),
		"-v", fmt.Sprintf("%s:%s", b.DockerSocketHostPath, b.DockerSocketHostPath),
		"-w", "/homelab",
		"-e", fmt.Sprintf("HOST_DATA_PATH=%s", b.ContainerMetadataHostPath),
		"--env-file", b.ContainerEnvFilePath,
		"-v", fmt.Sprintf("%s:%s", b.ContainerMetadataHostPath, b.ContainerMetadataMountPath),
		"-v", fmt.Sprintf("%s:%s:ro", cfg.Git.PrivKeyHostPath, cfg.Git.PrivKeyMountPath),
		cfg.General.Image,
	}
}

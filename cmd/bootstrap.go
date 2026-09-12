package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"

	appconfig "github.com/asdf57/homelabc/internal/config"
	"github.com/spf13/cobra"
)

const (
	bootstrapContainerEnvFileKey = "bootstrap.container_env_file"
	bootstrapMountPathKey        = "bootstrap.container_mount_path"
	bootstrapDockerSocketPathKey = "bootstrap.docker_socket_path"
	bootstrapHostDataPathKey     = "bootstrap.host_data_path"
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

	bindBootstrapFlag(bootstrapContainerEnvFileKey, "container-env-file")
	bindBootstrapFlag(bootstrapMountPathKey, "mount-path")
	bindBootstrapFlag(bootstrapDockerSocketPathKey, "docker-socket-path")
	bindBootstrapFlag(bootstrapHostDataPathKey, "host-data-path")
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
	groups, err := resolveBootstrapGroups(runtime.GOOS)
	if err != nil {
		return err
	}

	args := bootstrapDockerArgs(cfg, groups)
	cmd := exec.CommandContext(ctx, "docker", args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run bootstrap container: %w", err)
	}

	return nil
}

func resolveBootstrapGroups(goos string) ([]int, error) {
	if goos == "darwin" {
		// Docker Desktop and OrbStack expose the forwarded Docker socket as
		// root:root inside the Linux container.
		return []int{0}, nil
	}
	if goos != "linux" {
		return nil, fmt.Errorf("bootstrap is not supported on %s", goos)
	}

	dockerGroup, err := resolveDockerGroup()
	if err != nil {
		return nil, fmt.Errorf("resolve Docker group: %w", err)
	}
	homelabGroup, err := resolveHomelabGroup()
	if err != nil {
		return nil, fmt.Errorf("resolve homelab group: %w", err)
	}
	return []int{dockerGroup, homelabGroup}, nil
}

func bootstrapDockerArgs(cfg appconfig.Config, groups []int) []string {
	b := cfg.Bootstrap
	args := []string{
		"run",
		"--rm",
		"-it",
		"--privileged",
		"--network", "host",
	}
	for _, group := range groups {
		args = append(args, "--group-add", strconv.Itoa(group))
	}
	args = append(args,
		"-v", fmt.Sprintf("%s:/var/run/docker.sock", b.DockerSocket),
		"-w", "/homelab",
		"-e", fmt.Sprintf("HOST_DATA_PATH=%s", b.HostDataPath),
		"--env-file", b.EnvFile,
		"-v", fmt.Sprintf("%s:%s", b.HostDataPath, b.MountPath),
		"-e", fmt.Sprintf("INVENTORY_PUBLICATION_GROUP=%s", "localhost-inventory"),
		"-e", fmt.Sprintf("CONTAINER_MODE=%s", "bootstrap"),
		cfg.General.Image,
	)
	return args
}

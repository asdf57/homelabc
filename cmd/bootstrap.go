package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	bootstrapContainerEnvFileKey = "bootstrap.container-env-file"
	bootstrapMountPathKey        = "bootstrap.mount-path"
	bootstrapDockerSocketPathKey = "bootstrap.docker-socket-path"
	bootstrapHostDataPathKey     = "bootstrap.host-data-path"
	bootstrapImageKey            = "bootstrap.image"
)

// bootstrapOptions contains the fully resolved options for the bootstrap
// container. Values can come from command-line flags, HOMELABC_* environment
// variables, or the homelabc config file. containerEnvFile is kept opaque and
// passed to Docker; homelabc never imports configuration from its contents.
type bootstrapOptions struct {
	containerEnvFile string
	mountPath        string
	dockerSocketPath string
	hostDataPath     string
	image            string
}

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Bootstrap the homelab environment",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := resolveBootstrapOptions(viper.GetViper())
		if err != nil {
			return err
		}

		return runBootstrap(opts)
	},
}

func init() {
	rootCmd.AddCommand(bootstrapCmd)

	flags := bootstrapCmd.Flags()
	flags.String("container-env-file", ".env", "environment file passed unchanged to the bootstrap container")
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

// resolveBootstrapOptions applies this precedence, from highest to lowest:
//
//   - command-line flag
//   - HOMELABC_BOOTSTRAP_* environment variable
//   - homelabc configuration file
//   - built-in default
func resolveBootstrapOptions(config *viper.Viper) (bootstrapOptions, error) {
	opts := bootstrapOptionsFromConfig(config)

	if err := validateBootstrapOptions(opts); err != nil {
		return bootstrapOptions{}, err
	}

	return opts, nil
}

func bootstrapOptionsFromConfig(config *viper.Viper) bootstrapOptions {
	setBootstrapDefaults(config)

	opts := bootstrapOptions{
		containerEnvFile: strings.TrimSpace(config.GetString(bootstrapContainerEnvFileKey)),
		mountPath:        strings.TrimSpace(config.GetString(bootstrapMountPathKey)),
		dockerSocketPath: strings.TrimSpace(config.GetString(bootstrapDockerSocketPathKey)),
		hostDataPath:     strings.TrimSpace(config.GetString(bootstrapHostDataPathKey)),
		image:            strings.TrimSpace(config.GetString(bootstrapImageKey)),
	}

	// Match arch-provisioner's current Makefile behavior when no distinct
	// container-side mount path is configured.
	if opts.mountPath == "" {
		opts.mountPath = opts.hostDataPath
	}

	return opts
}

func setBootstrapDefaults(config *viper.Viper) {
	config.SetDefault(bootstrapContainerEnvFileKey, ".env")
	config.SetDefault(bootstrapDockerSocketPathKey, "/var/run/docker.sock")
	config.SetDefault(bootstrapImageKey, "prov")
}

func validateBootstrapOptions(opts bootstrapOptions) error {
	if opts.containerEnvFile == "" {
		return fmt.Errorf("container env file cannot be empty")
	}
	if info, err := os.Stat(opts.containerEnvFile); err != nil {
		return fmt.Errorf("access container env file %q: %w", opts.containerEnvFile, err)
	} else if info.IsDir() {
		return fmt.Errorf("container env file %q is a directory", opts.containerEnvFile)
	}

	if opts.hostDataPath == "" {
		return fmt.Errorf(
			"host data path is required; set --host-data-path, HOMELABC_BOOTSTRAP_HOST_DATA_PATH, or bootstrap.host-data-path in the homelabc config",
		)
	}
	if !filepath.IsAbs(opts.hostDataPath) {
		return fmt.Errorf("host data path must be absolute: %q", opts.hostDataPath)
	}
	if info, err := os.Stat(opts.hostDataPath); err != nil {
		return fmt.Errorf("access host data path %q: %w", opts.hostDataPath, err)
	} else if !info.IsDir() {
		return fmt.Errorf("host data path %q is not a directory", opts.hostDataPath)
	}

	if opts.mountPath == "" || !filepath.IsAbs(opts.mountPath) {
		return fmt.Errorf("container mount path must be absolute: %q", opts.mountPath)
	}

	if opts.dockerSocketPath == "" {
		return fmt.Errorf("Docker socket path cannot be empty")
	}
	if info, err := os.Stat(opts.dockerSocketPath); err != nil {
		return fmt.Errorf("access Docker socket %q: %w", opts.dockerSocketPath, err)
	} else if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("%q is not a Unix socket", opts.dockerSocketPath)
	}

	if opts.image == "" {
		return fmt.Errorf("bootstrap image cannot be empty")
	}

	return nil
}

/*
	DOCKER_PRIV_OPTS = --rm -it \
		--privileged \
		--network host \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v /lib/modules:/lib/modules:ro \
		-v /proc:/proc \
		-v /sys:/sys \
		-v /dev:/dev \
		-w /homelab \
		--env-file $(BOOTSTRAP_ENV_FILE) \
		-v $(HOST_GIT_PROVISIONING_KEY_FILE):/etc/ssh/git_provisioning_key:ro \
		-v $(HOST_PROVISIONING_KEY_FILE):/etc/ssh/provisioning_key:ro \
		-v $(HOST_DROPLET_KEY_FILE):/etc/ssh/droplet_key:ro \
		-v $(HOST_GIT_PROVISIONING_KEY_FILE).pub:/etc/ssh/git_provisioning_key.pub:ro \
		-v $(HOST_PROVISIONING_KEY_FILE).pub:/etc/ssh/provisioning_key.pub:ro \
		-v $(HOST_DROPLET_KEY_FILE).pub:/etc/ssh/droplet_key.pub:ro \
		-v $(HOST_DATA_PATH):$(MOUNTED_DATA_PATH) \

	DOCKER_UNPRIV_BASE_OPTS = --rm -it \
		-w /homelab \
		--env-file $(BOOTSTRAP_ENV_FILE) \
		-v $(HOST_DATA_PATH):$(MOUNTED_DATA_PATH)
*/

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

func runBootstrap(opts bootstrapOptions) error {
	// run docker container
	ctx := context.Background()

	dockerGroup, err := resolveDockerGroup()
	if err != nil {
		return fmt.Errorf("resolve Docker group: %w", err)
	}

	homelabGroup, err := resolveHomelabGroup()
	if err != nil {
		return fmt.Errorf("resolve homelab group: %w", err)
	}

	cmd := exec.CommandContext(
		ctx,
		"docker",
		"run",
		"--rm",
		"-it",
		"--privileged",
		"--network", "host",
		"--group-add", fmt.Sprintf("%d", dockerGroup),
		"--group-add", fmt.Sprintf("%d", homelabGroup),
		"-v", fmt.Sprintf("%s:%s", opts.dockerSocketPath, opts.dockerSocketPath),
		"-w", "/homelab",
		"-e", fmt.Sprintf("HOST_DATA_PATH=%s", opts.hostDataPath),
		"--env-file", opts.containerEnvFile,
		"-v", fmt.Sprintf("%s:%s", opts.hostDataPath, opts.mountPath),
		opts.image,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run bootstrap container: %w", err)
	}

	return nil
}

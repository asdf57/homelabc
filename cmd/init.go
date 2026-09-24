package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	appconfig "github.com/asdf57/homelabc/internal/config"
	"github.com/spf13/cobra"
)

const (
	initEnvFileKey               = "init.env_file"
	initMountPathKey             = "init.mount_path"
	initDockerSocketKey          = "init.docker_socket"
	initDataPathKey              = "init.data_path"
	initInventoryCaptureGroupKey = "init.inventory_capture_group"
)

var (
	includeArtifacts bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the homelab",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if err := cfg.ValidateInit(); err != nil {
			return err
		}
		return runInit(cmd.Context(), cfg, includeArtifacts)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	flags := initCmd.Flags()
	flags.String("env-file", "~/.homelab-init", "secrets passed to the initialization container")
	flags.String("mount-path", "", "container path for persistent data")
	flags.String("docker-socket", "/var/run/docker.sock", "host Docker socket")
	flags.String("data-path", "", "host directory for persistent homelab data")
	flags.String("inventory-capture-group", "platform", "capture group containing platform configuration")
	flags.BoolVar(&includeArtifacts, "artifacts", false, "build and publish provisioning images")

	for key, name := range map[string]string{
		initEnvFileKey: "env-file", initMountPathKey: "mount-path",
		initDockerSocketKey: "docker-socket", initDataPathKey: "data-path",
		initInventoryCaptureGroupKey: "inventory-capture-group",
	} {
		cobra.CheckErr(settings.BindPFlag(key, flags.Lookup(name)))
	}
}

func runInit(ctx context.Context, cfg appconfig.Config, artifacts bool) error {
	groups, err := resolveInitGroups(runtime.GOOS)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "docker", initDockerArgs(cfg, groups, artifacts)...)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("initialize homelab: %w", err)
	}
	return reportInitHealth(ctx, cfg)
}

func resolveInitGroups(goos string) ([]int, error) {
	if goos == "darwin" {
		return []int{0}, nil
	}
	if goos != "linux" {
		return nil, fmt.Errorf("initialization is not supported on %s", goos)
	}
	groups := make([]int, 0, 2)
	for _, name := range []string{"docker", "homelab"} {
		group, err := user.LookupGroup(name)
		if err != nil {
			return nil, fmt.Errorf("required group %q does not exist", name)
		}
		gid, err := strconv.Atoi(group.Gid)
		if err != nil {
			return nil, fmt.Errorf("parse %s group ID: %w", name, err)
		}
		groups = append(groups, gid)
	}
	return groups, nil
}

func initDockerArgs(cfg appconfig.Config, groups []int, artifacts bool) []string {
	b := cfg.Init
	args := []string{"run", "--rm", "--privileged", "--network", "host"}
	for _, group := range groups {
		args = append(args, "--group-add", strconv.Itoa(group))
	}
	args = append(args,
		"-v", fmt.Sprintf("%s:/var/run/docker.sock", b.DockerSocket),
		"-v", "/lib/modules:/lib/modules:ro",
		"-w", "/homelab",
		"--env-file", b.EnvFile,
		"-v", fmt.Sprintf("%s:%s", b.DataPath, b.MountPath),
		"-e", fmt.Sprintf("HOST_DATA_PATH=%s", b.DataPath),
		"-e", fmt.Sprintf("MOUNT_DATA_PATH=%s", b.MountPath),
		"-e", fmt.Sprintf("INVENTORY_CAPTURE_GROUP=%s", b.InventoryCaptureGroup),
		"-e", fmt.Sprintf("STIGMERGY_API_URL=%s", cfg.General.StigmergyApiUrl),
		"-e", fmt.Sprintf("HOMELAB_IMAGE=%s", cfg.General.Image),
		"-e", fmt.Sprintf("GIT_ANSIBLE_ROLES_REPO=%s", cfg.General.AnsibleRolesRepo),
		"-e", fmt.Sprintf("GIT_ANSIBLE_ROLES_REF=%s", cfg.General.AnsibleRolesRef),
		"-e", fmt.Sprintf("GIT_STIGMERGY_REPO=%s", b.StigmergyRepo),
		"-e", fmt.Sprintf("GIT_STIGMERGY_REF=%s", b.StigmergyRef),
		"-e", fmt.Sprintf("GIT_HOMELAB_INIT_REPO=%s", b.HomelabInitRepo),
		"-e", fmt.Sprintf("GIT_HOMELAB_INIT_REF=%s", b.HomelabInitRef),
		"-e", "CONTAINER_MODE=init",
		cfg.General.Image,
		"bash", "--login", "-c", initPlaybookCommand(artifacts),
	)
	return args
}

func initPlaybookCommand(artifacts bool) string {
	command := "set -e; ansible-playbook /homelab/plays/init.yml"
	if artifacts {
		command += "; ansible-playbook /homelab/plays/init_artifacts.yml"
	}
	return command
}

func reportInitHealth(ctx context.Context, cfg appconfig.Config) error {
	client := &http.Client{Timeout: 10 * time.Second}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(cfg.General.StigmergyApiUrl, "/")+"/readyz", nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("verify Stigmergy readiness: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("verify Stigmergy readiness: %s", response.Status)
	}

	composeFile := filepath.Join(cfg.Init.DataPath, "docker-compose.yml")
	cmd := exec.CommandContext(ctx, "docker", "compose", "--env-file", cfg.Init.EnvFile, "-p", "infra", "-f", composeFile, "ps")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

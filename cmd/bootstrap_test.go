package cmd

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	appconfig "github.com/asdf57/homelabc/internal/config"
)

func TestValidateBootstrapConfig(t *testing.T) {
	cfg := validBootstrapConfig(t)
	if err := cfg.ValidateBootstrap(); err != nil {
		t.Fatalf("ValidateBootstrap() error = %v", err)
	}

	tests := []struct {
		name    string
		change  func(*appconfig.Config)
		wantErr string
	}{
		{
			name:    "relative host path",
			change:  func(cfg *appconfig.Config) { cfg.Bootstrap.HostDataPath = "data" },
			wantErr: "host data path must be absolute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invalid := cfg
			tt.change(&invalid)
			err := invalid.ValidateBootstrap()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ValidateBootstrap() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestBootstrapDockerArgsForDarwin(t *testing.T) {
	cfg := validBootstrapConfig(t)
	args := bootstrapDockerArgs(cfg, []int{0})
	joined := strings.Join(args, " ")

	for _, want := range []string{
		"--group-add 0",
		cfg.Bootstrap.DockerSocket + ":/var/run/docker.sock",
		"MOUNT_DATA_PATH=" + cfg.Bootstrap.MountPath,
		"STIGMERGY_API_URL=http://stigmergy.example:8080",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("Docker arguments do not contain %q: %v", want, args)
		}
	}
}

func TestBootstrapDockerArgsOmitsEmptyStigmergyAPIURL(t *testing.T) {
	cfg := validBootstrapConfig(t)
	cfg.General.StigmergyApiUrl = ""

	if got := strings.Join(bootstrapDockerArgs(cfg, []int{0}), " "); strings.Contains(got, "STIGMERGY_API_URL") {
		t.Fatalf("bootstrapDockerArgs() contains an empty Stigmergy API URL: %s", got)
	}
}

func TestResolveBootstrapGroupsForDarwin(t *testing.T) {
	groups, err := resolveBootstrapGroups("darwin")
	if err != nil {
		t.Fatalf("resolveBootstrapGroups() error = %v", err)
	}
	if len(groups) != 1 || groups[0] != 0 {
		t.Fatalf("resolveBootstrapGroups() = %v, want [0]", groups)
	}
}

func validBootstrapConfig(t *testing.T) appconfig.Config {
	t.Helper()

	tempDir, err := os.MkdirTemp("/tmp", "homelabc-")
	if err != nil {
		t.Fatalf("create temporary directory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })
	envFile := filepath.Join(tempDir, "container.env")
	if err := os.WriteFile(envFile, nil, 0o600); err != nil {
		t.Fatalf("write %s: %v", envFile, err)
	}

	hostDataPath := filepath.Join(tempDir, "data")
	if err := os.Mkdir(hostDataPath, 0o755); err != nil {
		t.Fatalf("create host data directory: %v", err)
	}

	dockerSocketPath := filepath.Join(tempDir, "docker.sock")
	listener, err := net.Listen("unix", dockerSocketPath)
	if err != nil {
		t.Fatalf("create Docker socket: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	var cfg appconfig.Config
	cfg.Bootstrap.EnvFile = envFile
	cfg.Bootstrap.HostDataPath = hostDataPath
	cfg.Bootstrap.MountPath = "/homelab-data"
	cfg.Bootstrap.DockerSocket = dockerSocketPath
	cfg.General.Image = "prov"
	cfg.General.StigmergyApiUrl = "http://stigmergy.example:8080"
	return cfg
}

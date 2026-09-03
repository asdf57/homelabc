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
		{
			name:    "missing private key",
			change:  func(cfg *appconfig.Config) { cfg.Git.PrivateKey = "" },
			wantErr: "Git private key is required",
		},
		{
			name:    "relative private key mount",
			change:  func(cfg *appconfig.Config) { cfg.Git.PrivateKeyMount = "git_key" },
			wantErr: "Git private key mount path must be absolute",
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

func TestBootstrapDockerArgsMountsGitPrivateKeyReadOnly(t *testing.T) {
	cfg := validBootstrapConfig(t)
	args := bootstrapDockerArgs(cfg, 100, 200)
	want := cfg.Git.PrivateKey + ":" + cfg.Git.PrivateKeyMount + ":ro"

	for i := 0; i+1 < len(args); i++ {
		if args[i] == "-v" && args[i+1] == want {
			return
		}
	}
	t.Fatalf("Docker arguments do not contain %q: %v", want, args)
}

func validBootstrapConfig(t *testing.T) appconfig.Config {
	t.Helper()

	tempDir, err := os.MkdirTemp("/tmp", "homelabc-")
	if err != nil {
		t.Fatalf("create temporary directory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })
	envFile := filepath.Join(tempDir, "container.env")
	privateKey := filepath.Join(tempDir, "id_ed25519")
	for _, path := range []string{envFile, privateKey} {
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
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
	cfg.Git.PrivateKey = privateKey
	cfg.Git.PrivateKeyMount = "/etc/ssh/git_provisioning_key"
	cfg.General.Image = "prov"
	return cfg
}

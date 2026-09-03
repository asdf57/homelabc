package cmd

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/asdf57/homelabc/schemas"
)

func TestValidateBootstrapConfig(t *testing.T) {
	cfg := validBootstrapConfig(t)
	if err := validateBootstrapConfig(cfg); err != nil {
		t.Fatalf("validateBootstrapConfig() error = %v", err)
	}

	tests := []struct {
		name    string
		change  func(*schemas.Config)
		wantErr string
	}{
		{
			name:    "relative host path",
			change:  func(cfg *schemas.Config) { cfg.Bootstrap.ContainerMetadataHostPath = "data" },
			wantErr: "host data path must be absolute",
		},
		{
			name:    "missing private key",
			change:  func(cfg *schemas.Config) { cfg.Git.PrivKeyHostPath = "" },
			wantErr: "Git private key path is required",
		},
		{
			name:    "relative private key mount",
			change:  func(cfg *schemas.Config) { cfg.Git.PrivKeyMountPath = "git_key" },
			wantErr: "Git private key mount path must be absolute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invalid := cfg
			tt.change(&invalid)
			err := validateBootstrapConfig(invalid)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validateBootstrapConfig() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestBootstrapDockerArgsMountsGitPrivateKeyReadOnly(t *testing.T) {
	cfg := validBootstrapConfig(t)
	args := bootstrapDockerArgs(cfg, 100, 200)
	want := cfg.Git.PrivKeyHostPath + ":" + cfg.Git.PrivKeyMountPath + ":ro"

	for i := 0; i+1 < len(args); i++ {
		if args[i] == "-v" && args[i+1] == want {
			return
		}
	}
	t.Fatalf("Docker arguments do not contain %q: %v", want, args)
}

func validBootstrapConfig(t *testing.T) schemas.Config {
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

	var cfg schemas.Config
	cfg.Bootstrap.ContainerEnvFilePath = envFile
	cfg.Bootstrap.ContainerMetadataHostPath = hostDataPath
	cfg.Bootstrap.ContainerMetadataMountPath = "/homelab-data"
	cfg.Bootstrap.DockerSocketHostPath = dockerSocketPath
	cfg.Git.PrivKeyHostPath = privateKey
	cfg.Git.PrivKeyMountPath = "/etc/ssh/git_provisioning_key"
	cfg.General.Image = "prov"
	return cfg
}

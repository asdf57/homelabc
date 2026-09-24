package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	appconfig "github.com/asdf57/homelabc/internal/config"
)

func TestValidateInitConfig(t *testing.T) {
	cfg := validInitConfig(t)
	if err := cfg.ValidateInit(); err != nil {
		t.Fatalf("ValidateInit() error = %v", err)
	}
	cfg.Init.DataPath = "data"
	if err := cfg.ValidateInit(); err == nil || !strings.Contains(err.Error(), "data path must be absolute") {
		t.Fatalf("ValidateInit() error = %v", err)
	}
}

func TestInitDockerArgs(t *testing.T) {
	cfg := validInitConfig(t)
	joined := strings.Join(initDockerArgs(cfg, []int{0}, true), " ")
	for _, want := range []string{
		"--group-add 0", "INVENTORY_CAPTURE_GROUP=platform",
		"HOST_DATA_PATH=" + cfg.Init.DataPath,
		"STIGMERGY_API_URL=http://stigmergy.example:8080", "CONTAINER_MODE=init",
		"init.yml; ansible-playbook /homelab/plays/init_artifacts.yml",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("Docker arguments do not contain %q: %v", want, joined)
		}
	}
}

func validInitConfig(t *testing.T) appconfig.Config {
	t.Helper()
	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, "secrets.env")
	if err := os.WriteFile(envFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	dataPath := filepath.Join(tempDir, "data")
	if err := os.Mkdir(dataPath, 0o755); err != nil {
		t.Fatal(err)
	}
	return appconfig.Config{
		Init: appconfig.Init{
			EnvFile: envFile, DataPath: dataPath, MountPath: "/homelab-data",
			DockerSocket: "/var/run/docker.sock", InventoryCaptureGroup: "platform",
			StigmergyRepo: "https://example.test/stigmergy.git", StigmergyRef: "main",
			HomelabInitRepo: "https://example.test/homelab-init.git", HomelabInitRef: "main",
		},
		General: appconfig.General{
			Image: "homelab:latest", StigmergyApiUrl: "http://stigmergy.example:8080",
			AnsibleRolesRepo: "https://example.test/ansible-roles.git", AnsibleRolesRef: "main",
		},
	}
}

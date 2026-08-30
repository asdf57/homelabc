package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func TestBootstrapOptionsDoNotReadContainerEnv(t *testing.T) {
	tempDir := t.TempDir()
	hostDataPath := filepath.Join(tempDir, "data")
	if err := os.Mkdir(hostDataPath, 0o755); err != nil {
		t.Fatalf("create host data directory: %v", err)
	}

	containerEnvFile := filepath.Join(tempDir, "container.env")
	containerEnv := strings.Join([]string{
		"HOST_DATA_PATH=/must/not/be/used",
		"DOCKER_SOCKET=/must/not/be/used.sock",
		"IMAGE_NAME=must-not-be-used",
	}, "\n")
	if err := os.WriteFile(containerEnvFile, []byte(containerEnv), 0o600); err != nil {
		t.Fatalf("write container env file: %v", err)
	}

	dockerSocketPath := filepath.Join(tempDir, "configured-docker.sock")

	config := viper.New()
	config.Set(bootstrapContainerEnvFileKey, containerEnvFile)
	config.Set(bootstrapHostDataPathKey, hostDataPath)
	config.Set(bootstrapDockerSocketPathKey, dockerSocketPath)

	opts := bootstrapOptionsFromConfig(config)

	if opts.hostDataPath != hostDataPath {
		t.Fatalf("host data path = %q, want %q", opts.hostDataPath, hostDataPath)
	}
	if opts.mountPath != hostDataPath {
		t.Fatalf("mount path = %q, want host data path %q", opts.mountPath, hostDataPath)
	}
	if opts.dockerSocketPath != dockerSocketPath {
		t.Fatalf("Docker socket path = %q, want %q", opts.dockerSocketPath, dockerSocketPath)
	}
	if opts.imageName != "prov" {
		t.Fatalf("image name = %q, want built-in default %q", opts.imageName, "prov")
	}
	if opts.imageTag != "latest" {
		t.Fatalf("image tag = %q, want built-in default %q", opts.imageTag, "latest")
	}
}

func TestBootstrapOptionsUseViperPrecedence(t *testing.T) {
	tempDir := t.TempDir()
	hostDataPath := filepath.Join(tempDir, "data")
	if err := os.Mkdir(hostDataPath, 0o755); err != nil {
		t.Fatalf("create host data directory: %v", err)
	}

	containerEnvFile := filepath.Join(tempDir, "container.env")
	if err := os.WriteFile(containerEnvFile, nil, 0o600); err != nil {
		t.Fatalf("write container env file: %v", err)
	}

	dockerSocketPath := filepath.Join(tempDir, "docker.sock")

	config := viper.New()
	config.SetConfigType("yaml")
	configFile := strings.NewReader(`
bootstrap:
  image-tag: config-tag
`)
	if err := config.ReadConfig(configFile); err != nil {
		t.Fatalf("read test homelabc config: %v", err)
	}

	config.SetEnvPrefix("HOMELABC")
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	config.AutomaticEnv()
	t.Setenv("HOMELABC_BOOTSTRAP_IMAGE_TAG", "environment-tag")

	flags := pflag.NewFlagSet("bootstrap-test", pflag.ContinueOnError)
	flags.String("image-tag", "", "")
	if err := flags.Parse([]string{"--image-tag", "flag-tag"}); err != nil {
		t.Fatalf("parse test flags: %v", err)
	}
	if err := config.BindPFlag(bootstrapImageTagKey, flags.Lookup("image-tag")); err != nil {
		t.Fatalf("bind test flag: %v", err)
	}

	config.Set(bootstrapContainerEnvFileKey, containerEnvFile)
	config.Set(bootstrapHostDataPathKey, hostDataPath)
	config.Set(bootstrapDockerSocketPathKey, dockerSocketPath)

	opts := bootstrapOptionsFromConfig(config)

	if opts.imageTag != "flag-tag" {
		t.Fatalf("image tag = %q, want highest-precedence flag value", opts.imageTag)
	}
}

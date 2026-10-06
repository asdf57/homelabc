package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	appconfig "github.com/asdf57/homelabc/internal/config"
)

func TestValidateRunConfig(t *testing.T) {
	cfg := appconfig.Config{
		General: appconfig.General{
			Image:                 "prov",
			InventoryCaptureGroup: "servers",
			StigmergyApiUrl:       "http://stigmergy.example:8080",
			AnsibleRolesRepo:      "https://example.test/roles.git",
			AnsibleRolesRef:       "main",
		},
	}
	if err := cfg.ValidateRun(); err != nil {
		t.Fatalf("ValidateRun() error = %v", err)
	}

	tests := []struct {
		name    string
		change  func(*appconfig.Config)
		wantErr string
	}{
		{
			name:    "missing image",
			change:  func(cfg *appconfig.Config) { cfg.General.Image = "" },
			wantErr: "run image is required",
		},
		{
			name: "missing inventory capture group",
			change: func(cfg *appconfig.Config) {
				cfg.General.InventoryCaptureGroup = ""
			},
			wantErr: "inventory capture group is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invalid := cfg
			tt.change(&invalid)
			err := invalid.ValidateRun()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ValidateRun() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestRunnerCredentialMountsAreReadOnlyAndValidated(t *testing.T) {
	dir := t.TempDir()
	paths := []string{}
	for _, name := range []string{"token", "key", "cert", "known-hosts"} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	cfg := appconfig.Config{General: appconfig.General{Image: "runner", InventoryCaptureGroup: "servers", StigmergyApiUrl: "https://api.example", AnsibleRolesRepo: "https://example/roles", AnsibleRolesRef: "main", APITokenFile: paths[0], SSHPrivateKeyFile: paths[1], SSHCertificateFile: paths[2], SSHKnownHostsFile: paths[3]}}
	if err := cfg.ValidateRun(); err != nil {
		t.Fatal(err)
	}
	args := strings.Join(runDockerArgs(cfg), " ")
	for _, p := range paths {
		if !strings.Contains(args, "src="+p) {
			t.Fatal("missing credential mount")
		}
	}
	if strings.Count(args, ",readonly") != 4 {
		t.Fatal("credential mounts are not read-only")
	}
	cfg.General.SSHKnownHostsFile = ""
	if err := cfg.ValidateRun(); err != nil {
		t.Fatal("API-derived host trust must not require a manual known_hosts file:", err)
	}
	if strings.Contains(strings.Join(runDockerArgs(cfg), " "), "ANSIBLE_KNOWN_HOSTS_FILE") {
		t.Fatal("missing trust file was mounted")
	}
	cfg.General.SSHCertificateFile = ""
	if err := cfg.ValidateRun(); err == nil {
		t.Fatal("unpaired private key accepted")
	}
	cfg.General.SSHCertificateFile = paths[2]
	cfg.General.APITokenFile = "relative"
	if err := cfg.ValidateRun(); err == nil {
		t.Fatal("relative Docker mount accepted")
	}
}

func TestRunDockerArgs(t *testing.T) {
	cfg := appconfig.Config{
		General: appconfig.General{
			Image:                 "registry.example/homelab:v1",
			InventoryCaptureGroup: "production",
			StigmergyApiUrl:       "http://stigmergy.example:8080",
			AnsibleRolesRepo:      "https://example.test/roles.git",
			AnsibleRolesRef:       "stable",
		},
	}
	want := []string{
		"run",
		"--rm",
		"-it",
		"--network", "host",
		"-w", "/homelab",
		"-e", "INVENTORY_CAPTURE_GROUP=production",
		"-e", "GIT_ANSIBLE_ROLES_REPO=https://example.test/roles.git",
		"-e", "GIT_ANSIBLE_ROLES_REF=stable",
		"-e", "STIGMERGY_API_URL=http://stigmergy.example:8080",
		"-e", "CONTAINER_MODE=normal",
		"registry.example/homelab:v1",
	}

	if got := runDockerArgs(cfg); !reflect.DeepEqual(got, want) {
		t.Fatalf("runDockerArgs() = %v, want %v", got, want)
	}
}

func TestRunDockerArgsOmitsEmptyStigmergyAPIURL(t *testing.T) {
	cfg := appconfig.Config{
		General: appconfig.General{
			Image:                 "prov",
			InventoryCaptureGroup: "servers",
			AnsibleRolesRepo:      "https://example.test/roles.git",
			AnsibleRolesRef:       "main",
		},
	}

	if got := strings.Join(runDockerArgs(cfg), " "); strings.Contains(got, "STIGMERGY_API_URL") {
		t.Fatalf("runDockerArgs() contains an empty Stigmergy API URL: %s", got)
	}
}

func TestImageFlagIsSharedByRunAndInit(t *testing.T) {
	if runCmd.Flags().Lookup("image") != nil {
		t.Fatal("run command defines a local image flag")
	}
	if initCmd.Flags().Lookup("image") != nil {
		t.Fatal("init command defines a local image flag")
	}
	flag := rootCmd.PersistentFlags().Lookup("image")
	if flag == nil {
		t.Fatal("root command does not define the shared image flag")
	}
}

func TestStigmergyAPIURLFlagIsSharedByRunAndInit(t *testing.T) {
	if runCmd.Flags().Lookup("stigmergy-api-url") != nil {
		t.Fatal("run command defines a local Stigmergy API URL flag")
	}
	if initCmd.Flags().Lookup("stigmergy-api-url") != nil {
		t.Fatal("init command defines a local Stigmergy API URL flag")
	}
	flag := rootCmd.PersistentFlags().Lookup("stigmergy-api-url")
	if flag == nil {
		t.Fatal("root command does not define the shared Stigmergy API URL flag")
	}
}

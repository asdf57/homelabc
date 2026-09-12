package cmd

import (
	"reflect"
	"strings"
	"testing"

	appconfig "github.com/asdf57/homelabc/internal/config"
)

func TestValidateRunConfig(t *testing.T) {
	cfg := appconfig.Config{
		General: appconfig.General{
			Image:                     "prov",
			InventoryPublicationGroup: "servers-inventory",
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
			name: "missing inventory publication group",
			change: func(cfg *appconfig.Config) {
				cfg.General.InventoryPublicationGroup = ""
			},
			wantErr: "inventory publication group is required",
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

func TestRunDockerArgs(t *testing.T) {
	cfg := appconfig.Config{
		General: appconfig.General{
			Image:                     "registry.example/homelab:v1",
			InventoryPublicationGroup: "production-inventory",
		},
	}
	want := []string{
		"run",
		"--rm",
		"-it",
		"-w", "/homelab",
		"-e", "INVENTORY_PUBLICATION_GROUP=production-inventory",
		"-e", "CONTAINER_MODE=normal",
		"registry.example/homelab:v1",
	}

	if got := runDockerArgs(cfg); !reflect.DeepEqual(got, want) {
		t.Fatalf("runDockerArgs() = %v, want %v", got, want)
	}
}

func TestImageFlagIsSharedByRunAndBootstrap(t *testing.T) {
	if runCmd.Flags().Lookup("image") != nil {
		t.Fatal("run command defines a local image flag")
	}
	if bootstrapCmd.Flags().Lookup("image") != nil {
		t.Fatal("bootstrap command defines a local image flag")
	}
	flag := rootCmd.PersistentFlags().Lookup("image")
	if flag == nil {
		t.Fatal("root command does not define the shared image flag")
	}
}

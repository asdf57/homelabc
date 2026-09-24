package config

import "testing"

func TestExpandHome(t *testing.T) {
	cfg := Config{Init: Init{EnvFile: "~/.homelab-init"}}
	cfg.ExpandHome("/home/operator")
	if got, want := cfg.Init.EnvFile, "/home/operator/.homelab-init"; got != want {
		t.Fatalf("EnvFile = %q, want %q", got, want)
	}
}

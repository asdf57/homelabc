package config

import (
	"fmt"
	"os"
	"path/filepath"
)

func (c Config) ValidateInit() error {
	b := c.Init
	if err := validateFile("environment file", b.EnvFile, true); err != nil {
		return err
	}
	if err := validateDirectory("data path", b.DataPath); err != nil {
		return err
	}
	if !filepath.IsAbs(b.MountPath) {
		return fmt.Errorf("mount path must be absolute: %q", b.MountPath)
	}
	if err := validateSocket(b.DockerSocket); err != nil {
		return err
	}
	if c.General.Image == "" {
		return fmt.Errorf("image is required")
	}
	if c.General.StigmergyApiUrl == "" {
		return fmt.Errorf("Stigmergy API URL is required")
	}
	if b.InventoryCaptureGroup == "" {
		return fmt.Errorf("platform inventory capture group is required")
	}
	for name, value := range map[string]string{
		"Ansible roles repository": c.General.AnsibleRolesRepo,
		"Ansible roles revision":   c.General.AnsibleRolesRef,
		"Stigmergy repository":     c.Init.StigmergyRepo,
		"Stigmergy revision":       c.Init.StigmergyRef,
		"homelab-init repository":  c.Init.HomelabInitRepo,
		"homelab-init revision":    c.Init.HomelabInitRef,
	} {
		if value == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	return nil
}

func (c Config) ValidateRun() error {
	if c.General.Image == "" {
		return fmt.Errorf("run image is required")
	}
	if c.General.InventoryCaptureGroup == "" {
		return fmt.Errorf("inventory capture group is required")
	}
	if c.General.StigmergyApiUrl == "" {
		return fmt.Errorf("Stigmergy API URL is required")
	}
	if c.General.AnsibleRolesRepo == "" || c.General.AnsibleRolesRef == "" {
		return fmt.Errorf("Ansible roles repository and revision are required")
	}
	return nil
}

func validateFile(name, path string, absolute bool) error {
	if path == "" {
		return fmt.Errorf("%s is required", name)
	}
	if absolute && !filepath.IsAbs(path) {
		return fmt.Errorf("%s path must be absolute: %q", name, path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("access %s %q: %w", name, path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s %q is not a regular file", name, path)
	}
	return nil
}

func validateDirectory(name, path string) error {
	if path == "" {
		return fmt.Errorf("%s is required", name)
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("%s must be absolute: %q", name, path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("access %s %q: %w", name, path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s %q is not a directory", name, path)
	}
	return nil
}

func validateSocket(path string) error {
	if path == "" {
		return fmt.Errorf("Docker socket path is required")
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("access Docker socket %q: %w", path, err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("%q is not a Unix socket", path)
	}
	return nil
}

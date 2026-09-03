package config

import (
	"fmt"
	"os"
	"path/filepath"
)

func (c Config) ValidateBootstrap() error {
	b := c.Bootstrap
	if err := validateFile("container env file", b.EnvFile, false); err != nil {
		return err
	}
	if err := validateDirectory("host data path", b.HostDataPath); err != nil {
		return err
	}
	if !filepath.IsAbs(b.MountPath) {
		return fmt.Errorf("container mount path must be absolute: %q", b.MountPath)
	}
	if err := validateSocket(b.DockerSocket); err != nil {
		return err
	}
	if c.General.Image == "" {
		return fmt.Errorf("bootstrap image is required")
	}
	if !filepath.IsAbs(c.Git.PrivateKeyMount) {
		return fmt.Errorf("Git private key mount path must be absolute: %q", c.Git.PrivateKeyMount)
	}
	return validateFile("Git private key", c.Git.PrivateKey, true)
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

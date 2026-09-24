package config

import (
	"path/filepath"
	"strings"
)

type Config struct {
	Init    Init    `mapstructure:"init"`
	General General `mapstructure:"general"`
}

type Init struct {
	EnvFile               string `mapstructure:"env_file"`
	DataPath              string `mapstructure:"data_path"`
	MountPath             string `mapstructure:"mount_path"`
	DockerSocket          string `mapstructure:"docker_socket"`
	InventoryCaptureGroup string `mapstructure:"inventory_capture_group"`
	StigmergyRepo         string `mapstructure:"stigmergy_repo"`
	StigmergyRef          string `mapstructure:"stigmergy_ref"`
	HomelabInitRepo       string `mapstructure:"homelab_init_repo"`
	HomelabInitRef        string `mapstructure:"homelab_init_ref"`
}

type General struct {
	Image                 string `mapstructure:"image"`
	InventoryCaptureGroup string `mapstructure:"inventory_capture_group"`
	StigmergyApiUrl       string `mapstructure:"stigmergy_url"`
	AnsibleRolesRepo      string `mapstructure:"ansible_roles_repo"`
	AnsibleRolesRef       string `mapstructure:"ansible_roles_ref"`
}

func (c *Config) ApplyDefaults() {
	if c.Init.MountPath == "" {
		c.Init.MountPath = c.Init.DataPath
	}
	if c.Init.InventoryCaptureGroup == "" {
		c.Init.InventoryCaptureGroup = "platform"
	}
	if c.General.InventoryCaptureGroup == "" {
		c.General.InventoryCaptureGroup = "servers"
	}
	if c.General.AnsibleRolesRepo == "" {
		c.General.AnsibleRolesRepo = "https://github.com/asdf57/ansible-roles.git"
	}
	if c.General.AnsibleRolesRef == "" {
		c.General.AnsibleRolesRef = "main"
	}
	if c.Init.StigmergyRepo == "" {
		c.Init.StigmergyRepo = "https://github.com/asdf57/stigmergy.git"
	}
	if c.Init.StigmergyRef == "" {
		c.Init.StigmergyRef = "main"
	}
	if c.Init.HomelabInitRepo == "" {
		c.Init.HomelabInitRepo = "https://github.com/asdf57/homelab-init.git"
	}
	if c.Init.HomelabInitRef == "" {
		c.Init.HomelabInitRef = "main"
	}
}

func (c *Config) ExpandHome(home string) {
	if c.Init.EnvFile == "~" {
		c.Init.EnvFile = home
	} else if strings.HasPrefix(c.Init.EnvFile, "~/") {
		c.Init.EnvFile = filepath.Join(home, strings.TrimPrefix(c.Init.EnvFile, "~/"))
	}
}

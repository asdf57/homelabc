package config

type Config struct {
	Bootstrap Bootstrap `mapstructure:"bootstrap"`
	General   General   `mapstructure:"general"`
}

type Bootstrap struct {
	EnvFile      string `mapstructure:"container_env_file"`
	HostDataPath string `mapstructure:"host_data_path"`
	MountPath    string `mapstructure:"container_mount_path"`
	DockerSocket string `mapstructure:"docker_socket_path"`
}

type General struct {
	Image                     string `mapstructure:"image"`
	InventoryPublicationGroup string `mapstructure:"inventory_publication_group"`
}

func (c *Config) ApplyDefaults() {
	if c.Bootstrap.MountPath == "" {
		c.Bootstrap.MountPath = c.Bootstrap.HostDataPath
	}
	if c.General.InventoryPublicationGroup == "" {
		c.General.InventoryPublicationGroup = "servers-inventory"
	}
}

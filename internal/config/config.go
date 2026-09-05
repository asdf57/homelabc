package config

type Config struct {
	Bootstrap Bootstrap `mapstructure:"bootstrap"`
	Git       Git       `mapstructure:"git"`
	General   General   `mapstructure:"general"`
}

type Bootstrap struct {
	EnvFile                   string `mapstructure:"container_env_file"`
	HostDataPath              string `mapstructure:"host_data_path"`
	MountPath                 string `mapstructure:"container_mount_path"`
	DockerSocket              string `mapstructure:"docker_socket_path"`
	InventoryPublicationGroup string `mapstructure:"inventory_publication_group"`
}

type Git struct {
	PrivateKey      string `mapstructure:"priv_key_path"`
	PrivateKeyMount string `mapstructure:"container_priv_key_path"`
}

type General struct {
	Image string `mapstructure:"image"`
}

func (c *Config) ApplyDefaults() {
	if c.Bootstrap.MountPath == "" {
		c.Bootstrap.MountPath = c.Bootstrap.HostDataPath
	}
	if c.Bootstrap.InventoryPublicationGroup == "" {
		c.Bootstrap.InventoryPublicationGroup = "servers-inventory"
	}
	if c.Git.PrivateKeyMount == "" {
		c.Git.PrivateKeyMount = "/etc/ssh/git_provisioning_key"
	}
}

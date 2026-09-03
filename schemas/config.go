package schemas

type Config struct {
	Bootstrap struct {
		ContainerEnvFilePath       string `mapstructure:"container_env_file"`
		ContainerMetadataHostPath  string `mapstructure:"host_data_path"`
		ContainerMetadataMountPath string `mapstructure:"container_mount_path"`
		DockerSocketHostPath       string `mapstructure:"docker_socket_path"`
	} `mapstructure:"bootstrap"`

	Git struct {
		PrivKeyHostPath  string `mapstructure:"priv_key_path"`
		PrivKeyMountPath string `mapstructure:"container_priv_key_path"`
	} `mapstructure:"git"`

	General struct {
		Image string `mapstructure:"image"`
	} `mapstructure:"general"`
}

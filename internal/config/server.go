package config

type ServerConfig struct {
	Host string `json:"host"`
	Port uint16 `json:"port"`
}

func defaultServerConfig() ServerConfig {
	return ServerConfig{
		Host: "localhost",
		Port: 8080,
	}
}

func (cfg *ServerConfig) merge(other *ServerConfig) {
	if other.Host != "" {
		cfg.Host = other.Host
	}

	if other.Port != 0 {
		cfg.Port = other.Port
	}
}

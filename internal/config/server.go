package config

type ServerConfig struct {
	Host string
	Port uint16
}

func defaultServerConfig() ServerConfig {
	return ServerConfig{
		Host: "localhost",
		Port: 8080,
	}
}

func (sc *ServerConfig) merge(other *ServerConfig) {
	if other.Host != "" {
		sc.Host = other.Host
	}

	if other.Port != 0 {
		sc.Port = other.Port
	}
}

type serverConfigFile struct {
	Host *string `json:"host"`
	Port *uint16 `json:"port"`
}

func transformServerConfigFile(dcf *serverConfigFile) ServerConfig {
	if dcf == nil {
		return ServerConfig{}
	}

	var host string
	if dcf.Host != nil {
		host = *dcf.Host
	}

	var port uint16
	if dcf.Port != nil {
		port = *dcf.Port
	}

	return ServerConfig{
		Host: host,
		Port: port,
	}
}

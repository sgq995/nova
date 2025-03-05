package config

type Config struct {
	Router  RouterConfig
	Codegen CodegenConfig
	Server  ServerConfig
}

func Default() Config {
	return Config{
		Router:  defaultRouterConfig(),
		Codegen: defaultCodegenConfig(),
		Server:  defaultServerConfig(),
	}
}

func (c *Config) merge(other *Config) {
	c.Router.merge(&other.Router)
	c.Codegen.merge(&other.Codegen)
	c.Server.merge(&other.Server)
}

type configFile struct {
	Router  *routerConfigFile  `json:"router"`
	Codegen *codegenConfigFile `json:"codegen"`
	Server  *serverConfigFile  `json:"dev"`
}

func transformConfigFile(cf *configFile) Config {
	return Config{
		Router:  transformRouterConfigFile(cf.Router),
		Codegen: transformCodegenConfigFile(cf.Codegen),
		Server:  transformServerConfigFile(cf.Server),
	}
}

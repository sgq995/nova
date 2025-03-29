package config

type Config struct {
	Codegen CodegenConfig `json:"codegen"`
	Router  RouterConfig  `json:"router"`
	Server  ServerConfig  `json:"server"`
	Watcher WatcherConfig `json:"watcher"`
}

func Default() Config {
	return Config{
		Codegen: defaultCodegenConfig(),
		Router:  defaultRouterConfig(),
		Server:  defaultServerConfig(),
		Watcher: defaultWatcherConfig(),
	}
}

func (cfg *Config) merge(other *Config) {
	cfg.Codegen.merge(&other.Codegen)
	cfg.Router.merge(&other.Router)
	cfg.Server.merge(&other.Server)
	cfg.Watcher.merge(&other.Watcher)
}

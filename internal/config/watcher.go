package config

type WatcherConfig struct {
	Discovery int `json:"discovery"`
	Sync      int `json:"sync"`
}

func defaultWatcherConfig() WatcherConfig {
	return WatcherConfig{
		Discovery: 250,
		Sync:      500,
	}
}

func (cfg *WatcherConfig) merge(other *WatcherConfig) {
	if other.Discovery != 0 {
		cfg.Discovery = other.Discovery
	}

	if other.Sync != 0 {
		cfg.Sync = other.Sync
	}
}

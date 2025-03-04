package config

type RouterConfig struct {
	Pages string // relative path to pages dir, it defaults to "src/pages"
}

func defaultRouterConfig() RouterConfig {
	return RouterConfig{
		Pages: "src/pages",
	}
}

func (rc *RouterConfig) merge(other *RouterConfig) {
	if other.Pages != "" {
		rc.Pages = other.Pages
	}
}

type routerConfigFile struct {
	Pages string `json:"pages"`
}

func transformRouterConfigFile(rcf *routerConfigFile) RouterConfig {
	return RouterConfig{
		Pages: rcf.Pages,
	}
}

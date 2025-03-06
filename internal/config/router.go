package config

type RouterConfig struct {
	Pages   string // relative path to pages dir, it defaults to "src/pages"
	APIBase string // base path for rest routes, it defaults to "/api"
}

func defaultRouterConfig() RouterConfig {
	return RouterConfig{
		Pages:   "src/pages",
		APIBase: "/api",
	}
}

func (rc *RouterConfig) merge(other *RouterConfig) {
	if other.Pages != "" {
		rc.Pages = other.Pages
	}

	if other.APIBase != "" {
		rc.APIBase = other.APIBase
	}
}

type routerConfigFile struct {
	Pages   *string `json:"pages"`
	APIBase *string `json:"apiBase"`
}

func transformRouterConfigFile(rcf *routerConfigFile) RouterConfig {
	if rcf == nil {
		return RouterConfig{}
	}

	var pages string
	if rcf.Pages != nil {
		pages = *rcf.Pages
	}

	var apiBase string
	if rcf.APIBase != nil {
		apiBase = *rcf.APIBase
	}

	return RouterConfig{
		Pages:   pages,
		APIBase: apiBase,
	}
}

package config

import "path/filepath"

type RouterConfig struct {
	Src  string `json:"src"` // relative path to pages dir, it defaults to "src/pages"
	Http string `json:"http"`
}

func defaultRouterConfig() RouterConfig {
	return RouterConfig{
		Src:  filepath.FromSlash("src"),
		Http: filepath.FromSlash("internal/http"),
	}
}

func (cfg *RouterConfig) merge(other *RouterConfig) {
	if other.Src != "" {
		cfg.Src = filepath.FromSlash(other.Src)
	}

	if other.Http != "" {
		cfg.Http = filepath.FromSlash(other.Http)
	}
}

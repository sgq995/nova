package config

import "path/filepath"

type CodegenConfig struct {
	OutDir string `json:"outDir"` // relative path to the output directory, it defaults to ".nova"
}

func defaultCodegenConfig() CodegenConfig {
	return CodegenConfig{
		OutDir: filepath.FromSlash(".nova"),
	}
}

func (cfg *CodegenConfig) merge(other *CodegenConfig) {
	if other.OutDir != "" {
		cfg.OutDir = filepath.FromSlash(other.OutDir)
	}
}

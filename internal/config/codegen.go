package config

type CodegenConfig struct {
	OutDir string // relative path to the output directory, it defaults to ".nova"
}

func defaultCodegenConfig() CodegenConfig {
	return CodegenConfig{
		OutDir: ".nova",
	}
}

func (cc *CodegenConfig) merge(other *CodegenConfig) {
	if other.OutDir != "" {
		cc.OutDir = other.OutDir
	}
}

type codegenConfigFile struct {
	outDir *string
}

func transformCodegenConfigFile(ccf *codegenConfigFile) CodegenConfig {
	var outDir string
	if ccf.outDir != nil {
		outDir = *ccf.outDir
	}

	return CodegenConfig{
		OutDir: outDir,
	}
}

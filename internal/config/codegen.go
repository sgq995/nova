package config

type CodegenConfig struct {
	OutDir string // relative path to the output directory, it defaults to ".nova"
}

func defaultCodegenConfig() CodegenConfig {
	return CodegenConfig{
		OutDir: ".nova",
	}
}

func (*CodegenConfig) merge(other *CodegenConfig) {

}

type codegenConfigFile struct{}

func transformCodegenConfigFile(ccf *codegenConfigFile) CodegenConfig {
	return defaultCodegenConfig()
}

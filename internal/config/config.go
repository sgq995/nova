package config

import (
	"bytes"
	"encoding/json"
	"os"
)

type Config struct {
	Router  RouterConfig
	Codegen CodegenConfig
}

func Default() Config {
	return Config{
		Router: defaultRouterConfig(),
	}
}

func (c *Config) merge(other *Config) {
	c.Router.merge(&other.Router)
}

type configFile struct {
	Router  *routerConfigFile  `json:"router"`
	Codegen *codegenConfigFile `json:"codegen"`
}

func transformConfigFile(cf *configFile) Config {
	return Config{
		Router: transformRouterConfigFile(cf.Router),
	}
}

func readConfigFile(filename string) (Config, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}

	var cf configFile
	buf := bytes.NewReader(b)
	err = json.NewDecoder(buf).Decode(&cf)
	if err != nil {
		return Config{}, err
	}

	config := transformConfigFile(&cf)
	return config, nil
}

func Read(filename string) (Config, error) {
	cf, err := readConfigFile(filename)
	if err != nil {
		return Config{}, err
	}

	config := Default()
	config.merge(&cf)

	return config, nil
}

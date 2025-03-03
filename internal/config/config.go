package config

import (
	"bytes"
	"encoding/json"
	"os"
)

type Config struct {
	Pages string // Route to pages dir
}

func Default() Config {
	return Config{
		Pages: "src/pages",
	}
}

type configFile struct {
	Pages *string
}

func transformConfigFile(f *configFile) Config {
	pages := ""
	if f.Pages != nil {
		pages = *f.Pages
	}

	return Config{
		Pages: pages,
	}
}

func readConfigFile(filename string) (Config, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}

	var f configFile
	buf := bytes.NewReader(b)
	json.NewDecoder(buf).Decode(&f)

	config := transformConfigFile(&f)

	return config, nil
}

func override(base Config, config Config) Config {
	var final Config = base

	if config.Pages != "" {
		final.Pages = config.Pages
	}

	return final
}

func Read(filename string) (Config, error) {
	def := Default()

	user, err := readConfigFile(filename)
	if err != nil {
		return Config{}, err
	}

	config := override(def, user)
	return config, nil
}

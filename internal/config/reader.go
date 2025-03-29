package config

import (
	"bytes"
	"encoding/json"
	"os"
)

func readConfigFile(filename string) (Config, error) {
	var cfg Config

	b, err := os.ReadFile(filename)
	if err != nil {
		return cfg, err
	}

	buf := bytes.NewReader(b)
	err = json.NewDecoder(buf).Decode(&cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}

func Read(filename string) (Config, error) {
	cfg, err := readConfigFile(filename)
	if err != nil {
		return Config{}, err
	}

	config := Default()
	config.merge(&cfg)

	return config, nil
}

package config

import (
	"bytes"
	"encoding/json"
	"os"
)

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

package config

import (
	"encoding/json"
	"github.com/kelseyhightower/envconfig"
	"os"
)

func ReadConfig(path string) (AppConfig, error) {

	f, err := os.Open(path)
	if err != nil {
		return AppConfig{}, err
	}
	defer f.Close()

	var cfg AppConfig
	decoder := json.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return AppConfig{}, err
	}

	err = envconfig.Process("", &cfg)
	if err != nil {
		return AppConfig{}, err
	}

	return cfg, nil
}

package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Origins []ConfigOrigin `yaml:"origins"`
}

type ConfigOrigin struct {
	Origin        string `yaml:"origin"`
	Host          string `yaml:"host"`
	DisableUpdate bool   `yaml:"disableUpdate"`
	Rules         []Rule `yaml:"rules"`
}

func getConfig(filename string) (Config, error) {
	var config Config
	configBytes, err := os.ReadFile(filename)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(configBytes, &config)
	return config, err
}

package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port     int            `yaml:"port"`
	Provider string         `yaml:"provider"`
	Origins  []ConfigOrigin `yaml:"origins"`
}

type ConfigOrigin struct {
	Origin        string         `yaml:"origin"`
	Host          string         `yaml:"host"`
	DisableUpdate bool           `yaml:"disableUpdate"`
	Defaults      ConfigDefaults `yaml:"defaults"`
	Paths         []ConfigPath   `yaml:"paths"`
}

type ConfigDefaults struct {
	CacheControl string   `yaml:"cacheControl"`
	SafeMethods  []string `yaml:"safeMethods"`
}

type ConfigPath struct {
	Prefix              string `yaml:"prefix"`
	DefaultCacheControl string `yaml:"defaultCacheControl"`
}

func getConfig(filename string) (Config, error) {
	var config Config
	configBytes, err := os.ReadFile(filename)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(configBytes, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

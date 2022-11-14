package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type ConfigFile struct {
	Port     int      `yaml:"port"`
	Provider string   `yaml:"provider"`
	Origins  []Origin `yaml:"origins"`
}

type Origin struct {
	Origin              string   `yaml:"origin"`
	Host                string   `yaml:"host"`
	DefaultCacheControl string   `yaml:"defaultCacheControl"`
	DisableUpdate       bool     `yaml:"disableUpdate"`
	SafeMethods         []string `yaml:"safeMethods"`
	Paths               []Path   `yaml:"paths"`
}

type Path struct {
	Prefix              string `yaml:"prefix"`
	DefaultCacheControl string `yaml:"defaultCacheControl"`
}

func getConfig(filename string) (ConfigFile, error) {
	var config ConfigFile
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

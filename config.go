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
	Origin        string       `yaml:"origin"`
	Host          string       `yaml:"host"`
	DisableUpdate bool         `yaml:"disableUpdate"`
	Defaults      Defaults     `yaml:"defaults"`
	Paths         []ConfigPath `yaml:"paths"`
}

type ConfigPath struct {
	Prefix              string `yaml:"prefix"`
	DefaultCacheControl string `yaml:"defaultCacheControl"`
}

func (m *SafeMethods) UnmarshalYAML(value *yaml.Node) error {
	var arr []string
	err := value.Decode(&arr)
	if err != nil {
		return err
	}

	m.m = make(map[string]struct{})
	for _, method := range arr {
		m.m[method] = struct{}{}
	}

	return nil
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

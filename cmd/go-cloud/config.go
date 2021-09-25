package main

import (
	"os"

	storage "github.com/schoentoon/go-cloud/pkg/storage/utils"
	"gopkg.in/yaml.v2"
)

// Config structure of the config file
type Config struct {
	DB string `yaml:"db"`

	Storage storage.Config `yaml:"storage"`
}

// ReadConfig reads a file into the config structure
func ReadConfig(file string) (*Config, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	out := &Config{}
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&out)
	if err != nil {
		return nil, err
	}

	return out, err
}

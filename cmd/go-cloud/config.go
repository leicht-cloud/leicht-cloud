package main

import (
	"os"

	"github.com/schoentoon/go-cloud/pkg/auth"
	"github.com/schoentoon/go-cloud/pkg/fileinfo"
	"github.com/schoentoon/go-cloud/pkg/plugin"
	storage "github.com/schoentoon/go-cloud/pkg/storage/utils"
	"gopkg.in/yaml.v2"
)

// Config structure of the config file
type Config struct {
	Debug bool   `yaml:"debug"`
	DB    string `yaml:"db"`

	Storage  storage.Config  `yaml:"storage"`
	Plugin   plugin.Config   `yaml:"plugin"`
	FileInfo fileinfo.Config `yaml:"fileinfo"`
	Auth     auth.Config     `yaml:"auth"`
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

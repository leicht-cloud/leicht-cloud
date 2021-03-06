package main

import (
	"os"

	"github.com/leicht-cloud/leicht-cloud/pkg/app"
	"github.com/leicht-cloud/leicht-cloud/pkg/auth"
	"github.com/leicht-cloud/leicht-cloud/pkg/fileinfo"
	"github.com/leicht-cloud/leicht-cloud/pkg/plugin"
	"github.com/leicht-cloud/leicht-cloud/pkg/prometheus"
	storage "github.com/leicht-cloud/leicht-cloud/pkg/storage/utils"
	"gopkg.in/yaml.v2"
)

// Config structure of the config file
type Config struct {
	Debug    bool   `yaml:"debug"`
	DB       string `yaml:"db"`
	HttpAddr string `yaml:"addr"`

	Storage    storage.Config    `yaml:"storage"`
	Plugin     plugin.Config     `yaml:"plugin"`
	FileInfo   fileinfo.Config   `yaml:"fileinfo"`
	Apps       app.Config        `yaml:"apps"`
	Auth       auth.Config       `yaml:"auth"`
	Prometheus prometheus.Config `yaml:"prometheus"`
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

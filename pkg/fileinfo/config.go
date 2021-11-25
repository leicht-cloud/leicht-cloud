package fileinfo

import (
	"github.com/schoentoon/go-cloud/pkg/plugin"
	"github.com/schoentoon/go-cloud/pkg/prometheus"
)

type Config struct {
	MimeProvider string   `yaml:"mime_provider"`
	Providers    []string `yaml:"providers"`
}

func (c *Config) CreateProvider(pManager *plugin.Manager, prom *prometheus.Manager) (*Manager, error) {
	return NewManager(pManager, prom, c.MimeProvider, c.Providers...)
}

package app

import (
	"github.com/leicht-cloud/leicht-cloud/pkg/plugin"
	"github.com/leicht-cloud/leicht-cloud/pkg/prometheus"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
)

type Config struct {
	Apps []string `yaml:"apps"`
}

func (c *Config) CreateProvider(pManager *plugin.Manager, store storage.StorageProvider, prom *prometheus.Manager) (*Manager, error) {
	return NewManager(pManager, store, prom, c.Apps...)
}

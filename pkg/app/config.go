package app

import (
	"github.com/leicht-cloud/leicht-cloud/pkg/plugin"
	"github.com/leicht-cloud/leicht-cloud/pkg/prometheus"
)

type Config struct {
	Apps []string `yaml:"apps"`
}

func (c *Config) CreateProvider(pManager *plugin.Manager, prom *prometheus.Manager) (*Manager, error) {
	return NewManager(pManager, prom, c.Apps...)
}

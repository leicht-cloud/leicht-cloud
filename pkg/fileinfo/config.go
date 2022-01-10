package fileinfo

import "github.com/leicht-cloud/leicht-cloud/pkg/plugin"

type Config struct {
	MimeProvider string   `yaml:"mime_provider"`
	Providers    []string `yaml:"providers"`
}

func (c *Config) CreateProvider(pManager *plugin.Manager) (*Manager, error) {
	return NewManager(pManager, c.MimeProvider, c.Providers...)
}

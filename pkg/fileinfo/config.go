package fileinfo

type Config struct {
	MimeProvider string   `yaml:"mime_provider"`
	Providers    []string `yaml:"providers"`
}

func (c *Config) CreateProvider() (*Manager, error) {
	return NewManager(c.MimeProvider, c.Providers...)
}

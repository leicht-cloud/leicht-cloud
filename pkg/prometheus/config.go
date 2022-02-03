package prometheus

type Config struct {
	Enabled bool   `yaml:"enabled"`
	Address string `yaml:"address"`
}

func (c *Config) Create() (*Manager, error) {
	return newManager(c)
}

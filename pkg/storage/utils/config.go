package utils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/schoentoon/go-cloud/pkg/plugin"
	"github.com/schoentoon/go-cloud/pkg/storage"
	"github.com/schoentoon/go-cloud/pkg/storage/local"
	storagePlugin "github.com/schoentoon/go-cloud/pkg/storage/plugin"
)

type Config struct {
	Provider string                      `yaml:"provider"`
	Extra    map[interface{}]interface{} `yaml:"extra"`
}

func (c *Config) CreateProvider(pManager *plugin.Manager) (storage.StorageProvider, error) {
	return fromConfig(c, pManager)
}

func fromConfig(cfg *Config, pManager *plugin.Manager) (storage.StorageProvider, error) {
	if cfg.Provider == "local" {
		path, ok := cfg.Extra["path"]
		if ok {
			return &local.StorageProvider{RootPath: path.(string)}, nil
		}
		return nil, errors.New("No path provided for local storage provider?")
	} else if strings.HasPrefix(cfg.Provider, "plugin:") {
		name := strings.TrimPrefix(cfg.Provider, "plugin:")

		conn, err := pManager.Start(name)
		if err != nil {
			return nil, err
		}

		return storagePlugin.NewGrpcStorage(conn)
	}

	return nil, fmt.Errorf("No storage provider found with the name: %s", cfg.Provider)
}

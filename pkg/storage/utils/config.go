package utils

import (
	"errors"
	"fmt"
	"math/rand"
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

		// TODO: we'll want to communicate over a unix socket eventually
		// for now we just assign it to a random port.
		port := 60000 + (rand.Int31() % 5000)
		err := pManager.Start(name, port)
		if err != nil {
			return nil, err
		}

		return storagePlugin.NewGrpcStorage(fmt.Sprintf("127.0.0.1:%d", port))
	}

	return nil, fmt.Errorf("No storage provider found with the name: %s", cfg.Provider)
}

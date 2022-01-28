package utils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/leicht-cloud/leicht-cloud/pkg/plugin"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage/builtin/local"
	storagePlugin "github.com/leicht-cloud/leicht-cloud/pkg/storage/plugin"
)

type Config struct {
	Provider string                      `yaml:"provider"`
	Extra    map[interface{}]interface{} `yaml:"extra"`
}

func (c *Config) CreateProvider(pManager *plugin.Manager) (storage.StorageProvider, error) {
	out, err := fromConfig(c, pManager)
	if err != nil {
		return nil, err
	}
	return &ValidateWrapper{Next: out}, nil
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

		plugin, err := pManager.Start(name)
		if err != nil {
			return nil, err
		}

		conn, err := plugin.GrpcConn()
		if err != nil {
			return nil, err
		}

		return storagePlugin.NewGrpcStorage(conn, cfg.Extra)
	}

	return nil, fmt.Errorf("No storage provider found with the name: %s", cfg.Provider)
}

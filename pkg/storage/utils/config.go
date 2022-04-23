package utils

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/leicht-cloud/leicht-cloud/pkg/plugin"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage/builtin/local"
	storagePlugin "github.com/leicht-cloud/leicht-cloud/pkg/storage/plugin"
	"github.com/sirupsen/logrus"
	"go.uber.org/multierr"
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

		plugin, err := pManager.Start(name, "storage")
		if err != nil {
			return nil, err
		}

		conn, err := plugin.GrpcConn()
		if err != nil {
			return nil, err
		}

		store, err := storagePlugin.NewGrpcStorage(conn, cfg.Extra)
		if err != nil {
			stdout := plugin.StdoutDump()
			if len(stdout) > 0 {
				logrus.Infof("-----STDOUT FOR PLUGIN: %s-----", name)
				_, _ = io.Copy(os.Stdout, bytes.NewReader(stdout))
				logrus.Infof("-----END OF STDOUT FOR PLUGIN: %s-----", name)
			}

			closeErr := plugin.Close()

			return nil, multierr.Combine(err, closeErr)
		}

		return store, nil
	}

	return nil, fmt.Errorf("No storage provider found with the name: %s", cfg.Provider)
}

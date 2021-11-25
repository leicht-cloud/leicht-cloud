package prometheus

import (
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/schoentoon/go-cloud/pkg/fileinfo/types"
	"github.com/schoentoon/go-cloud/pkg/storage"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	gormprom "gorm.io/plugin/prometheus"
)

type Manager struct {
	enabled bool
	address string

	listener net.Listener

	promCheck, promRender *prometheus.SummaryVec
}

func newManager(cfg *Config) (*Manager, error) {
	if !cfg.Enabled { // if not enabled we just return a completely empty one
		return &Manager{}, nil
	}

	logrus.Info("Initializing prometheus")

	out := &Manager{
		enabled: true,
		address: cfg.Address,

		promCheck: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: "check",
			}, []string{"provider"},
		),
		promRender: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: "render",
			}, []string{"provider"},
		),
	}

	err := out.start()
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (m *Manager) start() error {
	if !m.enabled {
		return nil
	}

	listener, err := net.Listen("tcp", m.address)
	if err != nil {
		return err
	}
	m.listener = listener

	go func(listener net.Listener) {
		err = http.Serve(listener, promhttp.Handler())
		if err != nil {
			logrus.Error(err)
		}
	}(listener)

	registry := prometheus.WrapRegistererWithPrefix("fileinfo_", prometheus.DefaultRegisterer)

	registry.MustRegister(
		m.promCheck,
		m.promRender,
	)

	return nil
}

func (m *Manager) Close() error {
	if m.listener != nil {
		return m.listener.Close()
	}
	return nil
}

func (m *Manager) WrapStorage(store storage.StorageProvider, err error) (storage.StorageProvider, error) {
	if err != nil || !m.enabled {
		return store, err
	}

	return newWrappedStorage(store), nil
}

func (m *Manager) WrapFileInfo(fileinfo types.FileInfoProvider, name string) types.FileInfoProvider {
	if !m.enabled {
		return fileinfo
	}

	return m.newWrappedFileInfo(fileinfo, name)
}

func (m *Manager) WrapDB(db *gorm.DB) error {
	if !m.enabled {
		return nil
	}

	return db.Use(gormprom.New(gormprom.Config{}))
}

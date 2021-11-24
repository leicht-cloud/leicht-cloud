package prometheus

import (
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/schoentoon/go-cloud/pkg/storage"
	"github.com/sirupsen/logrus"
)

type Manager struct {
	enabled bool
	address string

	listener net.Listener
}

func newManager(cfg *Config) (*Manager, error) {
	if !cfg.Enabled { // if not enabled we just return a completely empty one
		return &Manager{}, nil
	}

	logrus.Info("Initializing prometheus")

	out := &Manager{
		enabled: true,
		address: cfg.Address,
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

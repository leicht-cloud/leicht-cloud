package plugin

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	prom "github.com/leicht-cloud/leicht-cloud/pkg/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type Manager struct {
	cfg *Config

	promManager   *prom.Manager
	runnerFactory RunnerFactory
	plugins       map[string]*plugin
}

type Config struct {
	Debug   bool                   `yaml:"debug"`
	Path    []string               `yaml:"path"`
	WorkDir string                 `yaml:"workdir"`
	Runner  string                 `yaml:"runtime"`
	Options map[string]interface{} `yaml:"options"`
}

func (c *Config) CreateManager(prom *prom.Manager) (*Manager, error) {
	runner, err := GetRunnerFactory(c.Runner)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(c.WorkDir, 0700)
	if err != nil {
		return nil, err
	}

	err = runner.configure(c.Options)
	if err != nil {
		return nil, err
	}

	return &Manager{
		cfg:           c,
		runnerFactory: runner,
		promManager:   prom,
		plugins:       make(map[string]*plugin),
	}, nil
}

func (m *Manager) Close() error {
	logrus.Info("Closing plugin manager")
	for name, plugin := range m.plugins {
		logrus.Infof("Closing %s", name)
		err := plugin.Close()
		if err != nil {
			logrus.Error(err)
		}
		logrus.Infof("Closed %s", name)
	}
	logrus.Info("Closed plugin manager")
	return nil
}

func (m *Manager) Plugins() []string {
	out := []string{}

	for plugin := range m.plugins {
		out = append(out, plugin)
	}

	return out
}

func (m *Manager) Stdout(name string) (*StdoutChannel, error) {
	plugin, ok := m.plugins[name]
	if !ok {
		return nil, fmt.Errorf("No plugin with the name: %s", name)
	}

	return plugin.stdout.Channel(), nil
}

const plugin_permissions = 0750

// The idea is that every single plugin will get their own working directory.
// As plugins can be packaged up we will copy the binary for the actual plugin
// into this working directory and execute it from there.
// We do however also support running against a directory directly, this does
// assume the binary is called the same as the plugin (it's copied regardless)
// and the directory has a manifest in it.
func (m *Manager) prepareDirectory(name, typ string) (*Manifest, error) {
	workDir := filepath.Join(m.cfg.WorkDir, name)
	err := os.MkdirAll(workDir, 0700)
	if err != nil {
		return nil, err
	}

	pluginFile := filepath.Join(workDir, "plugin")

	// TODO: We should improve where and how it searches for plugins
	// alongside we'll want to implement some form of manifest and
	// namespace the running plugin process

	for _, path := range m.cfg.Path {
		// PLUGIN FILE APPROACH //
		pluginPath := filepath.Join(path, fmt.Sprintf("%s.plugin", name))
		f, err := os.Open(pluginPath)
		if err == nil {
			defer f.Close()

			decompressor, err := gzip.NewReader(f)
			if err != nil {
				return nil, err
			}

			tr := tar.NewReader(decompressor)
			var manifest *Manifest
			copiedExe := false

			wantedExecutable := fmt.Sprintf("plugin-%s-%s", runtime.GOOS, runtime.GOARCH)

			for manifest == nil || !copiedExe {
				header, err := tr.Next()
				if err != nil {
					return nil, err
				}
				if header.Name == "plugin.manifest.yml" {
					manifest, err = parseManifest(tr)
					if err != nil {
						return nil, err
					}
					if manifest.Type != typ {
						return nil, fmt.Errorf("Incorrect type, got: %s, expected: %s", manifest.Type, typ)
					}
				} else if header.Name == wantedExecutable {
					dst, err := os.OpenFile(pluginFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, plugin_permissions)
					if err != nil {
						return nil, err
					}
					defer dst.Close()
					_, err = io.Copy(dst, tr)
					if err != nil {
						return nil, err
					}
					copiedExe = true
				}
			}

			if manifest != nil && copiedExe {
				return manifest, nil
			} else if manifest == nil {
				return nil, errors.New("Missing manifest")
			} else if !copiedExe {
				return nil, fmt.Errorf("Missing executable %s", wantedExecutable)
			}
		}

		// DIRECTORY APPROACH //
		// we first look if we have a build plugin in a directory with the same name
		dir := filepath.Join(path, name)
		fi, err := os.Stat(dir)
		if err == nil && fi.IsDir() {
			manifest, err := ParseManifestFromFile(filepath.Join(path, name))
			if err != nil {
				return nil, err
			}
			if manifest.Type != typ {
				return nil, fmt.Errorf("Incorrect type, got: %s, expected: %s", manifest.Type, typ)
			}
			src, err := os.Open(filepath.Join(dir, name))
			if err != nil {
				return nil, err
			}
			defer src.Close()
			dst, err := os.OpenFile(pluginFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, plugin_permissions)
			if err != nil {
				return nil, err
			}
			defer dst.Close()
			_, err = io.Copy(dst, src)
			return manifest, err
		}
	}

	return nil, fmt.Errorf("Plugin not found: %s", name)
}

func (m *Manager) Start(name, typ string) (PluginInterface, error) {
	manifest, err := m.prepareDirectory(name, typ)
	if err != nil {
		return nil, err
	}

	plugin, err := m.newPluginInstance(manifest, m.cfg, name)
	if err != nil {
		return nil, err
	}

	err = plugin.Start()
	if err != nil {
		return nil, err
	}

	if manifest.Prometheus {
		err = prometheus.Register(plugin)
		if err != nil {
			return nil, err
		}
	}

	m.plugins[name] = plugin

	return plugin, nil
}

package plugin

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type Manager struct {
	path    []string
	workDir string
	plugins []*exec.Cmd
}

type Config struct {
	Path    []string `yaml:"path"`
	WorkDir string   `yaml:"workdir"`
}

func (c *Config) CreateManager() (*Manager, error) {
	workDir := c.WorkDir
	err := os.MkdirAll(workDir, 0700)
	if err != nil {
		return nil, err
	}
	return &Manager{
		path:    c.Path,
		workDir: workDir,
		plugins: make([]*exec.Cmd, 0),
	}, nil
}

func (m *Manager) Close() error {
	for _, plugin := range m.plugins {
		err := plugin.Process.Signal(os.Interrupt)
		if err != nil {
			logrus.Errorf("Got %s, so killing it instead.", err)
			err = plugin.Process.Kill()
			if err != nil {
				logrus.Errorf("Error %s while killing process? wtf", err)
			}
		}
		// TODO: We should only wait for a certain time, don't give plugins infinite time to end cleanly
		err = plugin.Wait()
		if err != nil {
			logrus.Errorf("Error %s while waiting for %s to end", err, plugin)
		}
	}
	return nil
}

// The idea is that every single plugin will get their own working directory.
// As plugins can be packaged up we will copy the binary for the actual plugin
// into this working directory and execute it from there.
// We do however also support running against a directory directly, this does
// assume the binary is called the same as the plugin (it's copied regardless)
// and the directory has a manifest in it.
// TODO: run the plugin in namespaces and have said working directory actually be their root
func (m *Manager) prepareDirectory(name string) (*Manifest, error) {
	workDir := filepath.Join(m.workDir, name)
	err := os.MkdirAll(workDir, 0700)
	if err != nil {
		return nil, err
	}

	pluginFile := filepath.Join(workDir, "plugin")

	// TODO: We should improve where and how it searches for plugins
	// alongside we'll want to implement some form of manifest and
	// namespace the running plugin process

	for _, path := range m.path {
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
				} else if header.Name == wantedExecutable {
					dst, err := os.OpenFile(pluginFile, os.O_CREATE|os.O_WRONLY, 0500)
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
			src, err := os.Open(filepath.Join(dir, name))
			if err != nil {
				return nil, err
			}
			defer src.Close()
			dst, err := os.OpenFile(pluginFile, os.O_CREATE|os.O_WRONLY, 0500)
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

func (m *Manager) Start(name string) (*grpc.ClientConn, error) {
	workDir := filepath.Join(m.workDir, name)
	_, err := m.prepareDirectory(name)
	if err != nil {
		return nil, err
	}

	// TODO: Fall back to tcp on systems without support for unix sockets?
	socketFile := filepath.Join(workDir, "grpc.sock")

	cmd := exec.Cmd{
		Path: filepath.Join(workDir, "plugin"),
		Env:  []string{fmt.Sprintf("UNIXSOCKET=%s", socketFile)},
	}
	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	return grpc.Dial(fmt.Sprintf("unix://%s", socketFile),
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
}

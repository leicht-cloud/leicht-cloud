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
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/pkg/reexec"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type Manager struct {
	path       []string
	workDir    string
	namespaced bool
	plugins    map[string]*exec.Cmd
}

type Config struct {
	Path       []string `yaml:"path"`
	WorkDir    string   `yaml:"workdir"`
	Namespaced *bool    `yaml:"namespaced,omitempty"`
}

func (c *Config) CreateManager() (*Manager, error) {
	workDir := c.WorkDir
	err := os.MkdirAll(workDir, 0700)
	if err != nil {
		return nil, err
	}
	namespaced := true
	if c.Namespaced != nil {
		namespaced = *c.Namespaced
	}
	return &Manager{
		path:       c.Path,
		workDir:    workDir,
		namespaced: namespaced,
		plugins:    make(map[string]*exec.Cmd, 0),
	}, nil
}

func (m *Manager) Close() error {
	logrus.Info("Closing plugin manager")
	var wg sync.WaitGroup
	wg.Add(len(m.plugins))
	for name, plugin := range m.plugins {
		go func(wg *sync.WaitGroup, name string, plugin *exec.Cmd) {
			err := m.killProcess(name, plugin)
			if err != nil {
				logrus.Error(err)
			}
			wg.Done()
		}(&wg, name, plugin)
	}
	wg.Wait()
	logrus.Info("Closed plugin manager")
	return nil
}

func (m *Manager) killProcess(name string, plugin *exec.Cmd) error {
	// remove the unix socket
	defer os.Remove(filepath.Join(m.workDir, name, "grpc.sock"))

	err := plugin.Process.Signal(os.Interrupt)
	if err != nil {
		logrus.Errorf("Got %s, so killing it instead.", err)
		err = plugin.Process.Kill()
		if err != nil {
			return err
		}
	}
	c := make(chan error, 1)
	defer close(c)
	go func(c chan<- error, plugin *exec.Cmd) {
		c <- plugin.Wait()
	}(c, plugin)

	select {
	case err = <-c:
		return err
	case <-time.After(time.Second * 10):
		return plugin.Process.Kill()
	}
}

const plugin_permissions = 0750

// The idea is that every single plugin will get their own working directory.
// As plugins can be packaged up we will copy the binary for the actual plugin
// into this working directory and execute it from there.
// We do however also support running against a directory directly, this does
// assume the binary is called the same as the plugin (it's copied regardless)
// and the directory has a manifest in it.
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

func (m *Manager) Start(name string) (*grpc.ClientConn, error) {
	workDir := filepath.Join(m.workDir, name)
	_, err := m.prepareDirectory(name)
	if err != nil {
		return nil, err
	}

	// TODO: Fall back to tcp on systems without support for unix sockets?
	socketFile := filepath.Join(workDir, "grpc.sock")

	if m.namespaced {
		cmd := reexec.Command("pluginNamespace")
		cmd.Dir = workDir
		cmd.Env = []string{
			fmt.Sprintf("PLUGIN=%s", name),
		}
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWNS |
				syscall.CLONE_NEWUTS |
				syscall.CLONE_NEWIPC |
				syscall.CLONE_NEWPID |
				syscall.CLONE_NEWUSER,
			UidMappings: []syscall.SysProcIDMap{
				{
					ContainerID: 0,
					HostID:      os.Getuid(),
					Size:        1,
				},
			},
			GidMappings: []syscall.SysProcIDMap{
				{
					ContainerID: 0,
					HostID:      os.Getgid(),
					Size:        1,
				},
			},
		}
		err = cmd.Start()
		if err != nil {
			return nil, err
		}
		m.plugins[name] = cmd
	} else {
		cmd := exec.Cmd{
			Path: filepath.Join(workDir, "plugin"),
			Env: []string{
				fmt.Sprintf("UNIXSOCKET=%s", socketFile),
				fmt.Sprintf("PLUGIN=%s", name),
			},
		}
		err = cmd.Start()
		if err != nil {
			return nil, err
		}
		m.plugins[name] = &cmd
	}

	return grpc.Dial(fmt.Sprintf("unix://%s", socketFile),
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
}

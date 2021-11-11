package plugin

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	_ "github.com/schoentoon/go-cloud/pkg/plugin/namespace"

	"github.com/docker/docker/pkg/reexec"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type Manager struct {
	cfg *Config

	plugins map[string]*exec.Cmd
}

type Config struct {
	Debug      bool     `yaml:"debug"`
	Path       []string `yaml:"path"`
	WorkDir    string   `yaml:"workdir"`
	Namespaced *bool    `yaml:"namespaced,omitempty"`
	Bridge     struct {
		Enabled   bool   `yaml:"enabled"`
		Interface string `yaml:"interface"`
	} `yaml:"bridge"`
}

func (c *Config) CreateManager() (*Manager, error) {
	if c.Namespaced == nil {
		c.Namespaced = new(bool)
		*c.Namespaced = true
	}

	err := os.MkdirAll(c.WorkDir, 0700)
	if err != nil {
		return nil, err
	}

	return &Manager{
		cfg:     c,
		plugins: make(map[string]*exec.Cmd),
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
	defer os.Remove(filepath.Join(m.cfg.WorkDir, name, "grpc.sock"))

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

func (m *Manager) waitForSocket(socketFile string) error {
	maxWait := time.Second * 3
	checkInterval := time.Second
	timeStarted := time.Now()

	for {
		fi, err := os.Stat(socketFile)
		if err != nil {
			if os.IsNotExist(err) {
				if time.Since(timeStarted) > maxWait {
					return fmt.Errorf("Timeout after %s waiting for network", maxWait)
				}

				time.Sleep(checkInterval)
				continue
			}
			return err
		}
		if fi.Mode().Type() == os.ModeSocket {
			return nil
		}
		return fmt.Errorf("%s is not a unix socket??", socketFile)
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
	workDir := filepath.Join(m.cfg.WorkDir, name)
	manifest, err := m.prepareDirectory(name)
	if err != nil {
		return nil, err
	}

	// TODO: Fall back to tcp on systems without support for unix sockets?
	socketFile := filepath.Join(workDir, "grpc.sock")

	if *m.cfg.Namespaced {
		cmd := reexec.Command("pluginNamespace")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Dir = workDir
		cmd.Env = []string{
			fmt.Sprintf("PLUGIN=%s", name),
		}
		if m.cfg.Debug {
			cmd.Env = append(cmd.Env, "DEBUG=true")
		}
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWNS |
				syscall.CLONE_NEWUTS |
				syscall.CLONE_NEWIPC |
				syscall.CLONE_NEWPID |
				syscall.CLONE_NEWNET |
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

		if manifest.Permissions.Network {
			cmd.Env = append(cmd.Env, "NETWORK=true")
		}

		err = cmd.Start()
		if err != nil {
			return nil, err
		}
		m.plugins[name] = cmd

		if manifest.Permissions.Network {
			slirp := exec.Command("slirp4netns",
				"--enable-sandbox",
				"--enable-seccomp",
				"--enable-ipv6",
				strconv.Itoa(cmd.Process.Pid),
				"tap0")
			slirp.Stdout = os.Stdout
			slirp.Stderr = os.Stderr
			err = slirp.Start()
			if err != nil {
				logrus.Error(err)
			}
			// TODO: This is basically a hack, we should abstract this away neatly
			m.plugins[fmt.Sprintf("%s-slirp4netns", name)] = slirp
		}
	} else {
		cmd := exec.Cmd{
			Path: filepath.Join(workDir, "plugin"),
			Env: []string{
				fmt.Sprintf("UNIXSOCKET=%s", socketFile),
				fmt.Sprintf("PLUGIN=%s", name),
			},
		}
		if m.cfg.Debug {
			cmd.Env = append(cmd.Env, "DEBUG=true")
		}
		err = cmd.Start()
		if err != nil {
			return nil, err
		}
		m.plugins[name] = &cmd
	}

	err = m.waitForSocket(socketFile)
	if err != nil {
		return nil, err
	}

	return grpc.Dial(socketFile,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithReadBufferSize(0),
		grpc.WithWriteBufferSize(0),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", addr)
		}),
	)
}

package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/docker/docker/pkg/reexec"
)

const processKillTimeout = time.Second * 10

type plugin struct {
	workDir string
	process *exec.Cmd
}

func newPluginInstance(manifest *Manifest, cfg *Config, name string) (*plugin, error) {
	out := &plugin{
		workDir: filepath.Join(cfg.WorkDir, name),
	}
	if *cfg.Namespaced {
		cmd := reexec.Command("pluginNamespace")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Dir = out.workDir
		cmd.Env = []string{
			fmt.Sprintf("PLUGIN=%s", name),
		}
		if cfg.Debug {
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

		if manifest.Permissions.Network {
			// TODO: This is basically a hack, we should abstract this away neatly
		}
	} else {
		cmd := exec.Cmd{
			Path: filepath.Join(out.workDir, "plugin"),
			Env: []string{
				fmt.Sprintf("UNIXSOCKET=%s", out.SocketFile()),
				fmt.Sprintf("PLUGIN=%s", name),
			},
		}
		if cfg.Debug {
			cmd.Env = append(cmd.Env, "DEBUG=true")
		}
	}

	return nil, nil
}

func (p *plugin) SocketFile() string {
	return filepath.Join(p.workDir, "grpc.sock")
}

func (p *plugin) Start() error {
	return p.process.Start()
}

func (p *plugin) Close() error {
	// after closing we also remove the socketfile
	defer os.Remove(p.SocketFile())

	err := p.process.Process.Signal(os.Interrupt)
	if err != nil {
		err = p.process.Process.Kill()
		if err != nil {
			return err
		}
	}

	c := make(chan error, 1)
	defer close(c)
	go func(c chan<- error, process *exec.Cmd) {
		c <- process.Wait()
	}(c, p.process)

	select {
	case err := <-c:
		return err
	case <-time.After(processKillTimeout):
		return p.process.Process.Kill()
	}
}

func (p *plugin) waitForSocket() error {
	maxWait := time.Second * 3
	checkInterval := time.Second
	timeStarted := time.Now()
	socketFile := p.SocketFile()

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

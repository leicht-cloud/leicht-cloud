package plugin

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/docker/docker/pkg/reexec"
	"github.com/schoentoon/nsnet/pkg/host"
)

const processKillTimeout = time.Second * 10

type plugin struct {
	workDir string
	cmd     *exec.Cmd

	stdout bytes.Buffer

	nic *host.TunDevice
}

func newPluginInstance(manifest *Manifest, cfg *Config, name string) (*plugin, error) {
	p := &plugin{
		workDir: filepath.Join(cfg.WorkDir, name),
	}
	if *cfg.Namespaced {
		p.cmd = reexec.Command("pluginNamespace")
		p.cmd.Stdout = &p.stdout
		p.cmd.Stderr = &p.stdout
		p.cmd.Dir = p.workDir
		p.cmd.Env = []string{
			fmt.Sprintf("PLUGIN=%s", name),
		}
		if cfg.Debug {
			p.cmd.Env = append(p.cmd.Env, "DEBUG=true")
		}
		p.cmd.SysProcAttr = &syscall.SysProcAttr{
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

		// in the case of no network permission, we still go into a network namespace
		// with the difference that we never set up the network
		if manifest.Permissions.Network {
			p.cmd.Env = append(p.cmd.Env, fmt.Sprintf("NETWORK=%s", cfg.NetworkMode))

			if cfg.NetworkMode == "userspace" {
				tun, err := host.New(host.DefaultOptions())
				if err != nil {
					return nil, err
				}
				tun.AttachToCmd(p.cmd)

				p.nic = tun
			} else if cfg.NetworkMode == "host" {
				// if our network mode is host, we unset the CLONE_NEWNET flag so our containers run
				// with the actual host network
				p.cmd.SysProcAttr.Cloneflags = p.cmd.SysProcAttr.Cloneflags &^ syscall.CLONE_NEWNET
			} else {
				panic("we shouldn't be able to reach here")
			}
		}
	} else {
		p.cmd = &exec.Cmd{
			Stdout: &p.stdout,
			Stderr: &p.stdout,
			Path:   filepath.Join(p.workDir, "plugin"),
			Env: []string{
				fmt.Sprintf("UNIXSOCKET=%s", p.SocketFile()),
				fmt.Sprintf("PLUGIN=%s", name),
			},
		}
		if cfg.Debug {
			p.cmd.Env = append(p.cmd.Env, "DEBUG=true")
		}
	}

	return p, nil
}

func (p *plugin) SocketFile() string {
	return filepath.Join(p.workDir, "grpc.sock")
}

func (p *plugin) Start() error {
	return p.cmd.Start()
}

func (p *plugin) Close() error {
	// after closing we also remove the socketfile
	defer os.Remove(p.SocketFile())

	if p.nic != nil {
		defer p.nic.Close()
	}

	err := p.cmd.Process.Signal(os.Interrupt)
	if err != nil {
		err = p.cmd.Process.Kill()
		if err != nil {
			return err
		}
	}

	c := make(chan error, 1)
	defer close(c)
	go func(c chan<- error, process *exec.Cmd) {
		c <- process.Wait()
	}(c, p.cmd)

	select {
	case err := <-c:
		return err
	case <-time.After(processKillTimeout):
		return p.cmd.Process.Kill()
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
					return fmt.Errorf("Timeout after %s waiting for socket file %s", maxWait, socketFile)
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

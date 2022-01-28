package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/leicht-cloud/leicht-cloud/pkg/plugin/internal/namespace"

	"github.com/docker/docker/pkg/reexec"
	"github.com/schoentoon/nsnet/pkg/host"
	"github.com/sirupsen/logrus"
)

func init() {
	registerRunner("namespace", &namespaceFactory{})
}

type namespaceFactory struct {
	NetworkMode string
}

type namespaceRunner struct {
	cmd *exec.Cmd

	tun *host.TunDevice
}

func (n *namespaceFactory) configure(opts map[string]interface{}) error {
	raw, ok := opts["network_mode"]
	if !ok {
		logrus.Info("No network mode configured, defaulting to userspace")
		n.NetworkMode = "userspace"
		return nil
	}

	mode, ok := raw.(string)
	if !ok {
		return fmt.Errorf("network_mode specified isn't a valid string: %+v", raw)
	}

	switch mode {
	case "userspace":
		fallthrough
	case "host":
		n.NetworkMode = mode
	default:
		return fmt.Errorf("Invalid network_mode specified: %+v", mode)
	}

	return nil
}

func (n *namespaceFactory) Create(opts *RunOptions) (Runner, error) {
	out := &namespaceRunner{}

	out.cmd = reexec.Command("pluginNamespace")
	out.cmd.Stdout = opts.Stdout
	out.cmd.Stderr = opts.Stdout
	out.cmd.Dir = filepath.Join(opts.Config.WorkDir, opts.Name)
	out.cmd.Env = []string{
		fmt.Sprintf("PLUGIN=%s", opts.Name),
	}
	if opts.Config.Debug {
		out.cmd.Env = append(out.cmd.Env, "DEBUG=true")
	}
	out.cmd.SysProcAttr = &syscall.SysProcAttr{
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
	if opts.Manifest.Permissions.Network {
		out.cmd.Env = append(out.cmd.Env, fmt.Sprintf("NETWORK=%s", n.NetworkMode))

		if n.NetworkMode == "userspace" {
			tun, err := host.New(host.DefaultOptions())
			if err != nil {
				return nil, err
			}
			tun.AttachToCmd(out.cmd)

			out.tun = tun
		} else if n.NetworkMode == "host" {
			// if our network mode is host, we unset the CLONE_NEWNET flag so our containers run
			// with the actual host network
			out.cmd.SysProcAttr.Cloneflags = out.cmd.SysProcAttr.Cloneflags &^ syscall.CLONE_NEWNET
		} else {
			panic("we shouldn't be able to reach here")
		}
	}

	return out, nil
}

func (n *namespaceRunner) Start() error {
	return n.cmd.Start()
}

func (n *namespaceRunner) Close() error {
	err := n.cmd.Process.Signal(os.Interrupt)
	if err != nil {
		err = n.cmd.Process.Kill()
		if err != nil {
			return err
		}
	}

	c := make(chan error, 1)
	defer close(c)
	go func(c chan<- error, process *exec.Cmd) {
		c <- process.Wait()
	}(c, n.cmd)

	select {
	case err := <-c:
		return err
	case <-time.After(processKillTimeout):
		return n.cmd.Process.Kill()
	}
}

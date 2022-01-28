package plugin

import (
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/schoentoon/nsnet/pkg/host"
)

var network_modes = map[string]func() network{
	"host":        func() network { return &hostNet{} },
	"userspace":   func() network { return &userspaceNet{} },
	"slirp4netns": func() network { return &slirp4netns{} },
}

type network interface {
	PreStart(runner *namespaceRunner) error
	PostStart(runner *namespaceRunner) error
	PreClose(runner *namespaceRunner) error
	PostClose(runner *namespaceRunner) error
}

type hostNet struct {
}

func (h *hostNet) PreStart(runner *namespaceRunner) error {
	// we simply remove the CLONE_NEWNET flag to not namespace the network before starting
	runner.cmd.SysProcAttr.Cloneflags = runner.cmd.SysProcAttr.Cloneflags &^ syscall.CLONE_NEWNET
	return nil
}

func (h *hostNet) PostStart(runner *namespaceRunner) error {
	return nil
}

func (h *hostNet) PreClose(runner *namespaceRunner) error {
	return nil
}

func (h *hostNet) PostClose(runner *namespaceRunner) error {
	return nil
}

type userspaceNet struct {
	tun *host.TunDevice
}

func (u *userspaceNet) PreStart(runner *namespaceRunner) error {
	tun, err := host.New(host.DefaultOptions())
	if err != nil {
		return err
	}
	tun.AttachToCmd(runner.cmd)

	u.tun = tun
	return nil
}

func (u *userspaceNet) PostStart(runner *namespaceRunner) error {
	return nil
}

func (u *userspaceNet) PreClose(runner *namespaceRunner) error {
	return u.tun.Close()
}

func (u *userspaceNet) PostClose(runner *namespaceRunner) error {
	return nil
}

type slirp4netns struct {
	cmd *exec.Cmd
}

func (s *slirp4netns) PreStart(runner *namespaceRunner) error {
	return nil
}

func (s *slirp4netns) PostStart(runner *namespaceRunner) error {
	s.cmd = exec.Command("slirp4netns",
		"--enable-sandbox",
		"--enable-seccomp",
		"--mtu=65521",
		strconv.Itoa(runner.cmd.Process.Pid),
		"tap0",
	)
	s.cmd.Stdout = runner.cmd.Stdout
	s.cmd.Stderr = runner.cmd.Stderr

	return s.cmd.Start()
}

func (s *slirp4netns) PreClose(runner *namespaceRunner) error {
	return nil
}

func (s *slirp4netns) PostClose(runner *namespaceRunner) error {
	err := s.cmd.Process.Signal(os.Interrupt)
	if err != nil {
		err = s.cmd.Process.Kill()
		if err != nil {
			return err
		}
	}

	c := make(chan error, 1)
	defer close(c)
	go func(c chan<- error, process *exec.Cmd) {
		c <- process.Wait()
	}(c, s.cmd)

	select {
	case err := <-c:
		return err
	case <-time.After(processKillTimeout):
		return s.cmd.Process.Kill()
	}
}

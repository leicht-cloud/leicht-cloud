package plugin

import (
	"syscall"

	"github.com/schoentoon/nsnet/pkg/host"
)

var network_modes = map[string]func() network{
	"host":      func() network { return &hostNet{} },
	"userspace": func() network { return &userspaceNet{} },
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

package namespace

import (
	"net"
	"os"
	"os/exec"

	"github.com/docker/docker/pkg/reexec"
	"github.com/schoentoon/nsnet/pkg/container"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

func init() {
	reexec.Register("pluginNamespace", pluginNamespace)
	if reexec.Init() {
		os.Exit(0)
	}
}

func pluginNamespace() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	network := os.Getenv("NETWORK")

	if err := mountProc(wd); err != nil {
		panic(err)
	}

	if network == "userspace" {
		if err := container.MountTunDev(wd); err != nil {
			panic(err)
		}
	}

	if err := pivotRoot(wd); err != nil {
		panic(err)
	}

	switch network {
	case "userspace":
		{
			ifce, err := container.New()
			if err != nil {
				panic(err)
			}

			err = ifce.SetupNetwork()
			if err != nil {
				panic(err)
			}

			go func(ifce *container.TunDevice) {
				err := ifce.ReadLoop()
				if err != nil {
					logrus.Error(err)
				}
			}(ifce)
			go func(ifce *container.TunDevice) {
				err := ifce.WriteLoop()
				if err != nil {
					logrus.Error(err)
				}
			}(ifce)
		}
	case "slirp4netns":
		{
			// This assumes all the defaults of slirp4netns..
			link, err := netlink.LinkByName("tap0")
			if err != nil {
				panic(err)
			}

			addr := &netlink.Addr{
				IPNet: &net.IPNet{
					IP:   net.IPv4(10, 0, 2, 100),
					Mask: net.IPv4Mask(255, 255, 255, 0),
				},
			}
			err = netlink.AddrAdd(link, addr)
			if err != nil {
				panic(err)
			}

			err = netlink.LinkSetUp(link)
			if err != nil {
				panic(err)
			}

			route := &netlink.Route{
				Scope:     netlink.SCOPE_UNIVERSE,
				LinkIndex: link.Attrs().Index,
				Gw:        net.IPv4(10, 0, 2, 2),
			}
			err = netlink.RouteAdd(route)
			if err != nil {
				panic(err)
			}
		}
	}

	cmd := exec.Command("/plugin")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(),
		"UNIXSOCKET=/grpc.sock",
	)
	err = cmd.Run()
	if err != nil {
		panic(err)
	}
}

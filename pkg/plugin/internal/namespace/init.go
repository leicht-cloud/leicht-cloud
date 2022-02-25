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

var devices = []string{
	"/dev/null",
	"/dev/zero",
	"/dev/random",
	"/dev/urandom",
}

func init() {
	reexec.Register("pluginNamespace", pluginNamespace)
	if reexec.Init() {
		os.Exit(0)
	}
}

func pluginNamespace() {
	wd, err := os.Getwd()
	if err != nil {
		logrus.Panicf("Error during Getwd() call: %s", err)
	}

	network := os.Getenv("NETWORK")

	if err := mountProc(wd); err != nil {
		logrus.Panicf("Error in mountProc(): %s", err)
	}

	for _, device := range devices {
		if err := bindMount(wd, device); err != nil {
			logrus.Panicf("Error while binding %s: %s", device, err)
		}
	}

	if network == "userspace" {
		if err := container.MountTunDev(wd); err != nil {
			logrus.Panicf("Error in MountTunDev(): %s", err)
		}
	}

	// if we're either in userspace network mode or in host mode, we copy /etc/resolv.conf from the host for dns config
	// we copy this file as it is not uncommon for distros to have /etc/resolv.conf be a symlink to somewhere else on the host
	// system. In which case, bind mounting or whatever doesn't work. If this causes issues later on, we could always decide
	// to bind mount it if it's an actual file and only copy if it's a symlink or something alike I suppose.
	if network == "userspace" || network == "host" {
		if err := copyFile(wd, "/etc/resolv.conf"); err != nil {
			logrus.Panicf("Error while setting up /etc/resolv.conf: %s", err)
		}
	}

	if err := pivotRoot(wd); err != nil {
		logrus.Panicf("Error in pivotRoot(): %s", err)
	}

	switch network {
	case "userspace":
		{
			ifce, err := container.New()
			if err != nil {
				logrus.Panic(err)
			}

			err = ifce.SetupNetwork()
			if err != nil {
				logrus.Panic(err)
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
				logrus.Panic(err)
			}

			addr := &netlink.Addr{
				IPNet: &net.IPNet{
					IP:   net.IPv4(10, 0, 2, 100),
					Mask: net.IPv4Mask(255, 255, 255, 0),
				},
			}
			err = netlink.AddrAdd(link, addr)
			if err != nil {
				logrus.Panic(err)
			}

			err = netlink.LinkSetUp(link)
			if err != nil {
				logrus.Panic(err)
			}

			route := &netlink.Route{
				Scope:     netlink.SCOPE_UNIVERSE,
				LinkIndex: link.Attrs().Index,
				Gw:        net.IPv4(10, 0, 2, 2),
			}
			err = netlink.RouteAdd(route)
			if err != nil {
				logrus.Panic(err)
			}

			err = writeResolveConf("10.0.2.3")
			if err != nil {
				logrus.Panic(err)
			}
		}
	}

	cmd := exec.Command("/plugin")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(),
		"GRPC_UNIXSOCKET=/grpc.sock",
		"HTTP_UNIXSOCKET=/http.sock",
	)
	err = cmd.Run()
	if err != nil {
		logrus.Panic(err)
	}
}

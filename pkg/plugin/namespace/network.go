package namespace

import (
	"fmt"
	"net"
	"time"

	"github.com/vishvananda/netlink"
)

func waitForNetwork() error {
	maxWait := time.Second * 30
	checkInterval := time.Second
	timeStarted := time.Now()

	for {
		interfaces, err := net.Interfaces()
		if err != nil {
			return err
		}

		// pretty basic check ...
		// > 1 as a lo device will already exist
		if len(interfaces) > 1 {
			return nil
		}

		if time.Since(timeStarted) > maxWait {
			return fmt.Errorf("Timeout after %s waiting for network", maxWait)
		}

		time.Sleep(checkInterval)
	}
}

func setupNetwork() error {
	link, err := netlink.LinkByName("tap0")
	if err != nil {
		return err
	}

	addr := &netlink.Addr{
		IPNet: &net.IPNet{
			IP:   net.IPv4(10, 0, 2, 100),
			Mask: net.IPv4Mask(255, 255, 255, 0),
		},
	}
	err = netlink.AddrAdd(link, addr)
	if err != nil {
		return err
	}

	err = netlink.LinkSetUp(link)
	if err != nil {
		return err
	}

	route := &netlink.Route{
		Scope:     netlink.SCOPE_UNIVERSE,
		LinkIndex: link.Attrs().Index,
		Gw:        net.IPv4(10, 0, 2, 2),
	}
	return netlink.RouteAdd(route)
}

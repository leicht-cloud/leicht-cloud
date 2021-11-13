package namespace

import (
	"os"
	"os/exec"

	"github.com/docker/docker/pkg/reexec"
	"github.com/schoentoon/nsnet/pkg/container"
	"github.com/sirupsen/logrus"
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

	if network == "userspace" {
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

package namespace

import (
	"os"
	"os/exec"

	"github.com/docker/docker/pkg/reexec"
	"github.com/schoentoon/nsnet/pkg/container"
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

	network := os.Getenv("NETWORK") != ""

	if err := mountProc(wd); err != nil {
		panic(err)
	}

	if network {
		if err := container.MountTunDev(wd); err != nil {
			panic(err)
		}
	}

	if err := pivotRoot(wd); err != nil {
		panic(err)
	}

	if network {
		ifce, err := container.New()
		if err != nil {
			panic(err)
		}

		go ifce.ReadLoop()
		go ifce.WriteLoop()
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

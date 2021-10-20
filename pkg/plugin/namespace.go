package plugin

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/docker/docker/pkg/reexec"
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

	if err := mountProc(wd); err != nil {
		panic(err)
	}

	if err := pivotRoot(wd); err != nil {
		panic(err)
	}

	cmd := exec.Command("/plugin")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("UNIXSOCKET=/grpc.sock"),
	)
	err = cmd.Run()
	if err != nil {
		panic(err)
	}
}

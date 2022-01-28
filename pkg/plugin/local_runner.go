package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const processKillTimeout = time.Second * 10

func init() {
	registerRunner("local", &localFactory{})
}

type localFactory struct {
}

type local struct {
	cmd *exec.Cmd
}

func (l *localFactory) configure(opts map[string]interface{}) error {
	return nil
}

func (l *localFactory) Create(opts *RunOptions) (Runner, error) {
	out := &local{}
	out.cmd = &exec.Cmd{
		Stdout: opts.Stdout,
		Stderr: opts.Stdout,
		Path:   filepath.Join(opts.Config.WorkDir, opts.Name, "plugin"),
		Env: []string{
			fmt.Sprintf("UNIXSOCKET=%s", filepath.Join(opts.Config.WorkDir, opts.Name, "grpc.sock")),
			fmt.Sprintf("PLUGIN=%s", opts.Name),
		},
	}
	if opts.Config.Debug {
		out.cmd.Env = append(out.cmd.Env, "DEBUG=true")
	}

	return out, nil
}

func (l *local) Start() error {
	return l.cmd.Start()
}

func (l *local) Close() error {
	err := l.cmd.Process.Signal(os.Interrupt)
	if err != nil {
		err = l.cmd.Process.Kill()
		if err != nil {
			return err
		}
	}

	c := make(chan error, 1)
	defer close(c)
	go func(c chan<- error, process *exec.Cmd) {
		c <- process.Wait()
	}(c, l.cmd)

	select {
	case err := <-c:
		return err
	case <-time.After(processKillTimeout):
		return l.cmd.Process.Kill()
	}
}

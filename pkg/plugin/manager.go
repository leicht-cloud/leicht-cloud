package plugin

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func init() {
	rand.Seed(time.Now().Unix())
}

type Manager struct {
	Path string

	workDir string
	plugins []*exec.Cmd
}

func NewManager(pluginPath string) (*Manager, error) {
	workDir := "/tmp/go-cloud/" // TODO: Make this a config option
	err := os.MkdirAll(workDir, 0700)
	if err != nil {
		return nil, err
	}
	return &Manager{
		Path:    pluginPath,
		workDir: workDir,
		plugins: make([]*exec.Cmd, 0),
	}, nil
}

func (m *Manager) Close() error {
	for _, plugin := range m.plugins {
		err := plugin.Process.Signal(os.Interrupt)
		if err != nil {
			logrus.Errorf("Got %s, so killing it instead.", err)
			err = plugin.Process.Kill()
			if err != nil {
				logrus.Errorf("Error %s while killing process? wtf", err)
			}
		}
		// TODO: We should only wait for a certain time, don't give plugins infinite time to end cleanly
		err = plugin.Wait()
		if err != nil {
			logrus.Errorf("Error %s while waiting for %s to end", err, plugin)
		}
	}
	return nil
}

func (m *Manager) Start(name string) (*grpc.ClientConn, error) {
	// TODO: We should improve where and how it searches for plugins
	// alongside we'll want to implement some form of manifest and
	// namespace the running plugin process
	path := fmt.Sprintf("%s/%s/%s", m.Path, name, name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin '%s' not found at: %s", name, path)
	}

	// TODO: Fall back to tcp on systems without support for unix sockets?
	err := os.MkdirAll(filepath.Join(m.workDir, name), 0700)
	if err != nil {
		return nil, err
	}
	socketFile := filepath.Join(m.workDir, name, "grpc.sock")

	cmd := exec.Cmd{
		Path: path,
		Env:  []string{fmt.Sprintf("UNIXSOCKET=%s", socketFile)},
	}
	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	return grpc.Dial(fmt.Sprintf("unix://%s", socketFile),
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
}

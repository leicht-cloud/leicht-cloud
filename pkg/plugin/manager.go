package plugin

import (
	"fmt"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

type Manager struct {
	plugins []*exec.Cmd
}

func NewManager() *Manager {
	return &Manager{
		plugins: make([]*exec.Cmd, 0),
	}
}

func (m *Manager) Close() error {
	for _, plugin := range m.plugins {
		err := plugin.Process.Signal(os.Interrupt)
		if err != nil {
			log.Errorf("Got %s, so killing it instead.", err)
			plugin.Process.Kill()
		}
		plugin.Wait()
	}
	return nil
}

func (m *Manager) Start(name string, port int32) error {
	// TODO: We should improve where and how it searches for plugins
	// alongside we'll want to implement some form of manifest and
	// namespace the running plugin process
	path := fmt.Sprintf("./plugins/%s/%s", name, name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("plugin '%s' not found at: %s", name, path)
	}

	cmd := exec.Cmd{
		Path:   path,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Env:    []string{fmt.Sprintf("PORT=%d", port)},
	}
	err := cmd.Start()
	if err != nil {
		return err
	}
	return nil
}

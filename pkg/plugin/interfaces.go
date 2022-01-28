package plugin

import (
	"fmt"
)

var runners map[string]RunnerFactory = make(map[string]RunnerFactory)

type RunOptions struct {
	Name     string
	Config   *Config
	Manifest *Manifest
	Stdout   *Stdout
}

type RunnerFactory interface {
	configure(opts map[string]interface{}) error
	Create(opts *RunOptions) (Runner, error)
}

type Runner interface {
	Start() error
	Close() error
}

func registerRunner(name string, factory RunnerFactory) {
	runners[name] = factory
}

func GetRunnerFactory(name string) (RunnerFactory, error) {
	runner, ok := runners[name]
	if !ok {
		return nil, fmt.Errorf("No runner found with name: %s", name)
	}
	return runner, nil
}

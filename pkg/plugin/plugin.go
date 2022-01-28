package plugin

import (
	"context"
	"net"
	"path/filepath"

	"google.golang.org/grpc"
)

type PluginInterface interface {
	GrpcConn() (*grpc.ClientConn, error)
}

type plugin struct {
	workDir string

	runner Runner
	stdout *Stdout
}

func (m *Manager) newPluginInstance(manifest *Manifest, cfg *Config, name string) (*plugin, error) {
	p := &plugin{
		workDir: filepath.Join(cfg.WorkDir, name),
		stdout:  newStdout(),
	}

	runner, err := m.runnerFactory.Create(&RunOptions{
		Name:     name,
		Config:   cfg,
		Manifest: manifest,
		Stdout:   p.stdout,
	})
	if err != nil {
		return nil, err
	}
	p.runner = runner

	return p, nil
}

func (p *plugin) SocketFile() string {
	return filepath.Join(p.workDir, "grpc.sock")
}

func (p *plugin) Start() error {
	return p.runner.Start()
}

func (p *plugin) Close() error {
	return p.runner.Close()
}

func (p *plugin) GrpcConn() (*grpc.ClientConn, error) {
	return grpc.Dial(p.SocketFile(),
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithReadBufferSize(0),
		grpc.WithWriteBufferSize(0),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", addr)
		}),
	)
}

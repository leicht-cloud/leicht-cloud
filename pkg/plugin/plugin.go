package plugin

import (
	"context"
	"net"
	"net/http"
	"path/filepath"
	"time"

	prom "github.com/leicht-cloud/leicht-cloud/pkg/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type PluginInterface interface {
	GrpcConn() (*grpc.ClientConn, error)
	Stdout() []byte
	Close() error
}

type plugin struct {
	name    string
	workDir string

	httpClient http.Client

	runner Runner
	stdout *Stdout
}

func (m *Manager) newPluginInstance(manifest *Manifest, cfg *Config, name string) (*plugin, error) {
	p := &plugin{
		name:    name,
		workDir: filepath.Join(cfg.WorkDir, name),
		stdout:  newStdout(),
	}
	// we initialize httpClient seperate, as it needs an initialized plugin already for the httpSocketFile call
	p.httpClient = http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("unix", p.httpSocketFile())
			},
		},
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

func (p *plugin) grpcSocketFile() string {
	return filepath.Join(p.workDir, "grpc.sock")
}

func (p *plugin) httpSocketFile() string {
	return filepath.Join(p.workDir, "http.sock")
}

func (p *plugin) Start() error {
	return p.runner.Start()
}

func (p *plugin) Close() error {
	return p.runner.Close()
}

func (p *plugin) Stdout() []byte {
	return p.stdout.Bytes()
}

func (p *plugin) Describe(chan<- *prometheus.Desc) {
}

func (p *plugin) Collect(ch chan<- prometheus.Metric) {
	resp, err := p.httpClient.Get("http://localhost/metrics")
	if err != nil {
		logrus.Error(err)
		return
	}
	defer resp.Body.Close()

	var parser expfmt.TextParser
	out, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		logrus.Error(err)
		return
	}

	label := map[string]string{
		"plugin": p.name,
	}
	for _, parsedMetric := range out {
		metrics, err := prom.ParsedToMetric(label, parsedMetric)
		if err != nil {
			logrus.Error(err)
			continue
		}
		for _, metric := range metrics {
			ch <- metric
		}
	}
}

func (p *plugin) GrpcConn() (*grpc.ClientConn, error) {
	return grpc.Dial(p.grpcSocketFile(),
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithReadBufferSize(0),
		grpc.WithWriteBufferSize(0),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			logrus.Debugf("Connecting: %s", addr)
			var dialer net.Dialer
			dialer.Timeout = time.Second * 3
			return dialer.DialContext(ctx, "unix", addr)
		}),
	)
}

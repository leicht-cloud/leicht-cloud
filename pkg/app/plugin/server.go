package plugin

import (
	"context"
	"net"
	"time"

	"github.com/leicht-cloud/leicht-cloud/pkg/plugin/common"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
	storagePlugin "github.com/leicht-cloud/leicht-cloud/pkg/storage/plugin"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type App struct {
	storageConn *grpc.ClientConn
	storage     storage.StorageProvider
}

func Init() (*App, error) {
	return &App{}, nil
}

// This is meant to be called in the main() of your plugin
func (a *App) Loop() error {
	return common.Init(nil)
}

func (a *App) Close() error {
	if a.storageConn != nil {
		return a.storageConn.Close()
	}
	return nil
}

func (a *App) Storage() (storage.StorageProvider, error) {
	if a.storage != nil {
		return a.storage, nil
	}

	conn, err := grpc.Dial("storage.sock",
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

	if err != nil {
		return nil, err
	}

	a.storageConn = conn

	store, err := storagePlugin.NewGrpcStorage(conn, nil)
	if err != nil {
		return nil, err
	}
	a.storage = store

	return store, nil
}

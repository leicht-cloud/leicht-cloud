//go:generate protoc -I . ./storage.proto --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative

package plugin

import (
	"github.com/leicht-cloud/leicht-cloud/pkg/plugin/common"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
	grpc "google.golang.org/grpc"
)

// This is meant to be called in the main() of your plugin
func Start(storage storage.StorageProvider) (err error) {
	return common.Init(func(server *grpc.Server) error {
		RegisterStorageProviderServer(server, NewStorageBridge(storage))
		return nil
	})
}

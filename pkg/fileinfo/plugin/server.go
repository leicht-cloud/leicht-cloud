//go:generate protoc -I . ./fileinfo.proto --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative

package plugin

import (
	"github.com/leicht-cloud/leicht-cloud/pkg/fileinfo/types"
	"github.com/leicht-cloud/leicht-cloud/pkg/plugin/common"
	grpc "google.golang.org/grpc"
)

// This is meant to be called in the main() of your plugin
func Start(fileinfo types.FileInfoProvider) (err error) {
	return common.Init(func(server *grpc.Server) error {
		RegisterFileInfoProviderServer(server, NewFileinfoBridge(fileinfo))
		return nil
	})
}

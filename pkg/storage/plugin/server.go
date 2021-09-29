//go:generate protoc -I . ./service.proto --go_out=plugins=grpc:.

package plugin

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/schoentoon/go-cloud/pkg/storage"
	grpc "google.golang.org/grpc"
)

// This is meant to be called in the main() of your plugin
func Start(storage storage.StorageProvider) error {
	port := os.Getenv("PORT")
	lis, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%s", port))
	if err != nil {
		return err
	}
	log.Printf("Listening for grpc on %s\n", lis.Addr())

	server := grpc.NewServer(
		grpc.MaxRecvMsgSize(1024 * 1024 * 32),
	)
	RegisterStorageProviderServer(server, NewStorageBridge(storage))
	return server.Serve(lis)
}

//go:generate protoc -I . ./service.proto --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative

package plugin

import (
	"fmt"
	"net"
	"os"

	"github.com/schoentoon/go-cloud/pkg/storage"
	"github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"
)

// This is meant to be called in the main() of your plugin
func Start(storage storage.StorageProvider) (err error) {
	var lis net.Listener
	port := os.Getenv("PORT")
	unixSocket := os.Getenv("UNIXSOCKET")
	if unixSocket != "" {
		lis, err = net.Listen("unix", unixSocket)
	} else if port != "" {
		lis, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%s", port))
	} else {
		logrus.Fatalf("Neither PORT or UNIXSOCKET is specified")
	}
	if err != nil {
		return err
	}
	logrus.Infof("Listening for grpc on %s\n", lis.Addr())

	server := grpc.NewServer(
		grpc.MaxRecvMsgSize(1024 * 1024 * 32),
	)
	RegisterStorageProviderServer(server, NewStorageBridge(storage))
	return server.Serve(lis)
}

//go:generate protoc -I . ./storage.proto --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative

package plugin

import (
	"fmt"
	"net"
	"os"
	"os/signal"

	"github.com/schoentoon/go-cloud/pkg/storage"
	"github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"
)

// This is meant to be called in the main() of your plugin
func Start(storage storage.StorageProvider) (err error) {
	if os.Getenv("DEBUG") != "" {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.SetReportCaller(true)
	}
	var lis net.Listener
	port := os.Getenv("PORT")
	unixSocket := os.Getenv("UNIXSOCKET")
	if unixSocket != "" {
		lis, err = net.Listen("unix", unixSocket)
	} else if port != "" {
		lis, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%s", port))
	} else {
		logrus.Fatal("Neither PORT or UNIXSOCKET is specified")
	}
	if err != nil {
		return err
	}
	logrus.Infof("Listening for grpc on %s\n", lis.Addr())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	server := grpc.NewServer(
		grpc.MaxRecvMsgSize(1024*1024*32),
		grpc.WriteBufferSize(0),
		grpc.ReadBufferSize(0),
	)

	go func(server *grpc.Server, ch <-chan os.Signal) {
		<-c
		server.Stop()
		os.Exit(0)
	}(server, c)

	RegisterStorageProviderServer(server, NewStorageBridge(storage))
	return server.Serve(lis)
}

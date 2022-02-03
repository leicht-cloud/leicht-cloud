package common

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func createListener(prefix string) (net.Listener, error) {
	port := os.Getenv(fmt.Sprintf("%s_PORT", prefix))
	unixSocket := os.Getenv(fmt.Sprintf("%s_UNIXSOCKET", prefix))
	if unixSocket != "" {
		return net.Listen("unix", unixSocket)
	} else if port != "" {
		return net.Listen("tcp", fmt.Sprintf("127.0.0.1:%s", port))
	}
	return nil, fmt.Errorf("Neither %s_PORT or %s_UNIXSOCKET is specified", prefix, prefix)
}

func Init(registerGrpc func(*grpc.Server) error) error {
	if os.Getenv("DEBUG") != "" {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.SetReportCaller(true)
	}
	grpcListener, err := createListener("GRPC")
	if err != nil {
		return err
	}
	logrus.Infof("Listening on %s for grpc\n", grpcListener.Addr())

	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(1024*1024*32),
		grpc.WriteBufferSize(0),
		grpc.ReadBufferSize(0),
	)

	err = registerGrpc(grpcServer)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func(wg *sync.WaitGroup) {
		logrus.Info("Starting grpc listener")
		err := grpcServer.Serve(grpcListener)
		if err != nil {
			logrus.Error(err)
		}
		wg.Done()
	}(&wg)

	httpListener, err := createListener("HTTP")
	if err != nil {
		return err
	}

	httpServer := http.Server{
		Handler: promhttp.Handler(),
	}

	go func(wg *sync.WaitGroup) {
		logrus.Info("Starting prometheus listener")
		err := httpServer.Serve(httpListener)
		if err != nil {
			logrus.Error(err)
		}
		wg.Done()
	}(&wg)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func(grpcServer *grpc.Server, httpServer *http.Server, ch <-chan os.Signal) {
		<-c
		grpcServer.Stop()
		httpServer.Close()
		os.Exit(0)
	}(grpcServer, &httpServer, c)

	wg.Wait()

	return nil
}

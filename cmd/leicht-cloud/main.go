package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"

	gchttp "github.com/leicht-cloud/leicht-cloud/pkg/http"
	"github.com/leicht-cloud/leicht-cloud/pkg/models"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	var cfgfile = flag.String("config", "config.yml", "Config file location")
	flag.Parse()

	config, err := ReadConfig(*cfgfile)
	if err != nil {
		logrus.Fatal(err)
	}

	if config.Debug {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.SetReportCaller(true)
	}

	logrus.Infof("Initializing database: %+v", config.DB)
	db, err := gorm.Open(sqlite.Open(config.DB), &gorm.Config{})
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Info("Updating database models")
	err = models.InitModels(db)
	if err != nil {
		logrus.Fatal(err)
	}

	auth, err := config.Auth.Create(db)
	if err != nil {
		logrus.Fatal(err)
	}

	prom, err := config.Prometheus.Create()
	if err != nil {
		logrus.Fatal(err)
	}
	defer prom.Close()

	err = prom.WrapDB(db)
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Info("Initializing plugin manager")
	pluginManager, err := config.Plugin.CreateManager(prom)
	if err != nil {
		logrus.Fatal(err)
	}
	defer pluginManager.Close()

	logrus.Infof("Initializing storage provider %s", config.Storage.Provider)
	storage, err := prom.WrapStorage(config.Storage.CreateProvider(pluginManager))
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Infof("Initializing file info providers")
	fileinfo, err := config.FileInfo.CreateProvider(pluginManager, prom)
	if err != nil {
		logrus.Fatal(err)
	}
	defer fileinfo.Close()

	logrus.Infof("Initializing apps")
	apps, err := config.Apps.CreateProvider(pluginManager, storage, prom)
	if err != nil {
		logrus.Fatal(err)
	}
	defer apps.Close()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	logrus.Info("Initialing http server")
	server, err := gchttp.InitHttpServer(config.HttpAddr, db, auth, storage, pluginManager, apps, fileinfo)
	if err != nil {
		logrus.Fatal(err)
	}

	go func(server *http.Server, ch <-chan os.Signal) {
		<-c
		logrus.Info("Closing server")
		server.Close()
	}(server, c)

	err = server.ListenAndServe()
	if err != nil {
		logrus.Error(err)
	}
}

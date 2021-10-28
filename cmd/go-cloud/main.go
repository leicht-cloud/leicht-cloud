package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"

	"github.com/schoentoon/go-cloud/pkg/auth"
	gchttp "github.com/schoentoon/go-cloud/pkg/http"
	"github.com/schoentoon/go-cloud/pkg/models"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	var cfgfile = flag.String("config", "config.yml", "Config file location")
	flag.Parse()

	logrus.SetLevel(logrus.DebugLevel)

	config, err := ReadConfig(*cfgfile)
	if err != nil {
		logrus.Fatal(err)
	}

	db, err := gorm.Open(sqlite.Open(config.DB), &gorm.Config{})
	if err != nil {
		logrus.Fatal(err)
	}

	err = models.InitModels(db)
	if err != nil {
		logrus.Fatal(err)
	}

	auth := auth.NewProvider(db)

	pluginManager, err := config.Plugin.CreateManager()
	if err != nil {
		logrus.Fatal(err)
	}
	defer pluginManager.Close()

	storage, err := config.Storage.CreateProvider(pluginManager)
	if err != nil {
		logrus.Fatal(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	server, err := gchttp.InitHttpServer(db, auth, storage)
	if err != nil {
		logrus.Fatal(err)
	}

	go func(server *http.Server, ch <-chan os.Signal) {
		<-c
		server.Close()
	}(server, c)

	err = server.ListenAndServe()
	if err != nil {
		logrus.Error(err)
	}
}

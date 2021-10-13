package main

import (
	"flag"

	"github.com/schoentoon/go-cloud/pkg/auth"
	gchttp "github.com/schoentoon/go-cloud/pkg/http"
	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/plugin"

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
		panic(err)
	}

	db, err := gorm.Open(sqlite.Open(config.DB), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	err = models.InitModels(db)
	if err != nil {
		panic(err)
	}

	auth := auth.NewProvider(db)

	pluginManager, err := plugin.NewManager("./plugins")
	if err != nil {
		panic(err)
	}
	defer pluginManager.Close()

	storage, err := config.Storage.CreateProvider(pluginManager)
	if err != nil {
		panic(err)
	}

	server, err := gchttp.InitHttpServer(db, auth, storage)
	if err != nil {
		panic(err)
	}
	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

package main

import (
	"flag"

	"github.com/schoentoon/go-cloud/pkg/auth"
	gchttp "github.com/schoentoon/go-cloud/pkg/http"
	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/plugin"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	var cfgfile = flag.String("config", "config.yml", "Config file location")
	flag.Parse()

	log.SetLevel(log.DebugLevel)

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

	pluginManager := plugin.NewManager()
	defer pluginManager.Close()

	storage, err := config.Storage.CreateProvider(pluginManager)
	if err != nil {
		panic(err)
	}

	server, err := gchttp.InitHttpServer(db, auth, storage)
	if err != nil {
		panic(err)
	}
	server.ListenAndServe()
}

package main

import (
	"github.com/schoentoon/go-cloud/pkg/auth"
	gchttp "github.com/schoentoon/go-cloud/pkg/http"
	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/storage/local"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	log.SetLevel(log.DebugLevel)

	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	err = models.InitModels(db)
	if err != nil {
		panic(err)
	}

	auth := auth.NewProvider(db)

	storage := &local.StorageProvider{RootPath: "./tmp"}

	server, err := gchttp.InitHttpServer(db, auth, storage)
	if err != nil {
		panic(err)
	}
	server.ListenAndServe()
}

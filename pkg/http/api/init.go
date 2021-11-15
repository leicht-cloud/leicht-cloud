package api

import (
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/storage"
	"gorm.io/gorm"
)

func Init(mux *http.ServeMux, db *gorm.DB, storage storage.StorageProvider) {
	mux.Handle("/api/upload", newUploadHandler(storage))
	mux.Handle("/api/list", newListHandler(storage))
}

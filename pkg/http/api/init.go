package api

import (
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/auth"
	"github.com/schoentoon/go-cloud/pkg/storage"
	"gorm.io/gorm"
)

func Init(mux *http.ServeMux, db *gorm.DB, auth *auth.Provider, storage storage.StorageProvider) {
	mux.Handle("/api/upload", newUploadHandler(storage, auth))
	mux.Handle("/api/list", newListHandler(storage, auth))
}

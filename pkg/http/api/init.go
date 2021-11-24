package api

import (
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/fileinfo"
	"github.com/schoentoon/go-cloud/pkg/storage"
	"gorm.io/gorm"
)

func Init(mux *http.ServeMux, db *gorm.DB, storage storage.StorageProvider, fileinfo *fileinfo.Manager) {
	mux.Handle("/api/upload", newUploadHandler(storage))
	mux.Handle("/api/download", newDownloadHandler(storage))
	mux.Handle("/api/list", newListHandler(storage))
	mux.Handle("/api/fileinfo", newFileInfoHandler(storage, fileinfo))
}

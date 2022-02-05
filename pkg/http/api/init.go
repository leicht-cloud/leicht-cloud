package api

import (
	"net/http"

	"github.com/leicht-cloud/leicht-cloud/pkg/fileinfo"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
	"gorm.io/gorm"
)

func Init(mux *http.ServeMux, db *gorm.DB, storage storage.StorageProvider, fileinfo *fileinfo.Manager) {
	mux.Handle("/api/upload", newUploadHandler(db, storage))
	mux.Handle("/api/download", newDownloadHandler(db, storage))
	mux.Handle("/api/list", newListHandler(storage))
	mux.Handle("/api/fileinfo", newFileInfoHandler(storage, fileinfo))
	mux.Handle("/api/mkdir", newMkdirHandler(storage))
}

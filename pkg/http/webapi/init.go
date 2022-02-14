package webapi

import (
	"net/http"

	"github.com/leicht-cloud/leicht-cloud/pkg/fileinfo"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
	"gorm.io/gorm"
)

func Init(mux *http.ServeMux, db *gorm.DB, storage storage.StorageProvider, fileinfo *fileinfo.Manager) {
	mux.Handle("/webapi/upload", newUploadHandler(db, storage))
	mux.Handle("/webapi/download", newDownloadHandler(db, storage))
	mux.Handle("/webapi/list", newListHandler(storage))
	mux.Handle("/webapi/fileinfo", newFileInfoHandler(storage, fileinfo))
	mux.Handle("/webapi/mkdir", newMkdirHandler(storage))
	mux.Handle("/webapi/delete", newDeleteHandler(storage))
}

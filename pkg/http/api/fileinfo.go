package api

import (
	"encoding/json"
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/auth"
	"github.com/schoentoon/go-cloud/pkg/fileinfo"
	_ "github.com/schoentoon/go-cloud/pkg/fileinfo/builtin"
	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/storage"
	"github.com/sirupsen/logrus"
)

type fileInfoHandler struct {
	Storage  storage.StorageProvider
	FileInfo *fileinfo.Manager
}

func newFileInfoHandler(store storage.StorageProvider, fileinfo *fileinfo.Manager) http.Handler {
	return auth.AuthHandler(&fileInfoHandler{Storage: store, FileInfo: fileinfo})
}

func (h *fileInfoHandler) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")

	file, err := h.Storage.File(r.Context(), user, filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer file.Close()

	out, err := h.FileInfo.FileInfo(filename, file, &fileinfo.Options{Render: true}, "md5", "sha1")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(out)
	if err != nil {
		logrus.Error(err)
	}
}

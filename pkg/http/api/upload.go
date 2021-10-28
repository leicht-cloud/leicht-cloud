package api

import (
	"io"
	"net/http"
	"path"

	"github.com/schoentoon/go-cloud/pkg/auth"
	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/storage"
	"github.com/sirupsen/logrus"
)

type uploadHandler struct {
	Storage storage.StorageProvider
}

func newUploadHandler(store storage.StorageProvider, authProvider *auth.Provider) http.Handler {
	return auth.AuthHandler(authProvider, &uploadHandler{Storage: store})
}

func (h *uploadHandler) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	// TODO figure out a way to upload without having to read the entire thing in memory right away..
	err := r.ParseMultipartForm(1024 * 1024 * 8)
	if err != nil {
		http.Error(w, "Malformed request", http.StatusInternalServerError)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	f, err := h.Storage.File(r.Context(), user, path.Join("/", header.Filename))
	if err != nil {
		logrus.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer f.Close()
	_, err = io.Copy(f, file)
	if err != nil {
		logrus.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

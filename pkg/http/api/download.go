package api

import (
	"fmt"
	"io"
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/auth"
	_ "github.com/schoentoon/go-cloud/pkg/fileinfo/builtin"
	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/storage"
	"github.com/sirupsen/logrus"
)

type downloadHandler struct {
	Storage storage.StorageProvider
}

func newDownloadHandler(store storage.StorageProvider) http.Handler {
	return auth.AuthHandler(&downloadHandler{Storage: store})
}

func (h *downloadHandler) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")

	file, err := h.Storage.File(r.Context(), user, filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer file.Close()

	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	_, err = io.Copy(w, file)
	if err != nil {
		logrus.Error(err)
	}
}

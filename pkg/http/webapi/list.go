package webapi

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/leicht-cloud/leicht-cloud/pkg/auth"
	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
)

type listHandler struct {
	Storage storage.StorageProvider
}

func newListHandler(store storage.StorageProvider) http.Handler {
	return auth.AuthHandler(&listHandler{Storage: store})
}

func (h *listHandler) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("dir")
	if dir == "" {
		dir = "/"
	}

	files, err := h.Storage.ListDirectory(r.Context(), user, dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for file := range files {
		err = json.NewEncoder(w).Encode(file)
		if err != nil {
			logrus.Errorf("Error %s while encoding json", err)
		}
	}
}

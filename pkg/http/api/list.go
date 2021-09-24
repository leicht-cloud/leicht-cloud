package api

import (
	"encoding/json"
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/auth"
	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/storage"
)

type listHandler struct {
	Storage storage.StorageProvider
}

func newListHandler(store storage.StorageProvider, authProvider *auth.Provider) http.Handler {
	return auth.AuthHandler(authProvider, &listHandler{Storage: store})
}

func (h *listHandler) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	files, err := h.Storage.ListDirectory(user, "/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO error checking
	json.NewEncoder(w).Encode(files)
}

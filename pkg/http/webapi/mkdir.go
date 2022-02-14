package webapi

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/leicht-cloud/leicht-cloud/pkg/auth"
	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
)

type mkdirHandler struct {
	Storage storage.StorageProvider
}

func newMkdirHandler(store storage.StorageProvider) http.Handler {
	return auth.AuthHandler(&mkdirHandler{Storage: store})
}

func (h *mkdirHandler) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	path := r.Form.Get("path")
	foldername := r.Form.Get("foldername")
	if path == "" || foldername == "" {
		http.Error(w, "Invalid parameters", http.StatusBadRequest)
		return
	}

	dir := filepath.Join(path, foldername)

	err = h.Storage.Mkdir(r.Context(), user, dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/?dir=%s", dir), http.StatusTemporaryRedirect)
}

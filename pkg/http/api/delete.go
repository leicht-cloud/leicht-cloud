package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/leicht-cloud/leicht-cloud/pkg/auth"
	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
	"go.uber.org/multierr"
)

type deleteHandler struct {
	Storage storage.StorageProvider
}

func newDeleteHandler(store storage.StorageProvider) http.Handler {
	return auth.AuthHandler(&deleteHandler{Storage: store})
}

func (h *deleteHandler) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	files, ok := r.Form["file"]
	if !ok {
		http.Error(w, "Invalid parameters", http.StatusBadRequest)
		return
	}

	dir := ""
	for _, file := range files {
		if dir == "" {
			index := strings.LastIndex(file, "/")
			if index > 0 {
				dir = file[:index]
			}
		}

		delErr := h.Storage.Delete(r.Context(), user, file)
		if delErr != nil {
			err = multierr.Append(err, delErr)
		}
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/?dir=%s", dir), http.StatusTemporaryRedirect)
}

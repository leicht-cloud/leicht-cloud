package api

import (
	"fmt"
	"io"
	"net/http"

	"github.com/leicht-cloud/leicht-cloud/pkg/auth"
	_ "github.com/leicht-cloud/leicht-cloud/pkg/fileinfo/builtin"
	"github.com/leicht-cloud/leicht-cloud/pkg/http/helper/limiter"
	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type downloadHandler struct {
	Storage storage.StorageProvider
}

func newDownloadHandler(db *gorm.DB, store storage.StorageProvider) http.Handler {
	return auth.AuthHandler(
		limiter.DownloadMiddleware(db,
			&downloadHandler{
				Storage: store,
			},
		),
	)
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

package api

import (
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path"
	"strings"

	"github.com/schoentoon/go-cloud/pkg/auth"
	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/storage"
	"github.com/sirupsen/logrus"
)

type uploadHandler struct {
	Storage storage.StorageProvider
}

func newUploadHandler(store storage.StorageProvider) http.Handler {
	return auth.AuthHandler(&uploadHandler{Storage: store})
}

func (h *uploadHandler) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				logrus.Error(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			filename := p.FileName()
			if filename == "" {
				logrus.Error(err)
				http.Error(w, "Empty filename?", http.StatusBadRequest)
				return
			}

			f, err := h.Storage.File(r.Context(), user, path.Join("/", filename))
			if err != nil {
				logrus.Error(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer f.Close()

			_, err = io.Copy(f, p)
			if err != nil {
				logrus.Error(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		return
	}

	http.Error(w, "Invalid request, expected multipart", http.StatusBadRequest)
}

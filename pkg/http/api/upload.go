package api

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/schoentoon/go-cloud/pkg/auth"
	"github.com/schoentoon/go-cloud/pkg/http/helper/limiter"
	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/storage"
	"github.com/sirupsen/logrus"
)

type uploadHandler struct {
	Storage storage.StorageProvider

	mutex   sync.RWMutex
	nextID  int64
	uploads map[int64]*uploadState
}

type uploadState struct {
	UserID   uint64
	Length   uint64
	Position uint64
	File     storage.File
}

func newUploadHandler(store storage.StorageProvider) http.Handler {
	return limiter.Middleware(1024*100, 1024*100, auth.AuthHandler(
		&uploadHandler{
			Storage: store,
			uploads: make(map[int64]*uploadState),
		}),
	)
}

func (h *uploadHandler) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Add("Tus-Extension", "creation")
		w.Header().Add("Tus-Resumable", "1.0.0")
		w.Header().Add("Tus-Version", "1.0.0")
		w.WriteHeader(http.StatusOK)
		return
	} else if r.URL.Query().Has("resume") {
		// if we have the resume parameter it is an attempt to resume a previously started upload
		id, err := strconv.ParseInt(r.URL.Query().Get("resume"), 10, 64)
		if err != nil {
			http.Error(w, "Invalid id?", http.StatusBadRequest)
			return
		}

		h.mutex.RLock()
		state, ok := h.uploads[id]
		h.mutex.RUnlock()

		if !ok {
			logrus.Errorf("Couldn't find state with id: %d", id)
			http.Error(w, "No previous upload found with this id", http.StatusNotFound)
			return
		}

		done := state.Handle(w, r)
		if done {
			h.mutex.Lock()
			delete(h.uploads, id)
			h.mutex.Unlock()

			err = state.Close()
			if err != nil {
				logrus.Error(err)
			}
		}
		return
	} else if r.Method == http.MethodPost && r.Header.Get("Upload-Length") != "" {
		// POST & Upload-Length header indicates a create file request for the tus protocol
		length, err := strconv.ParseUint(r.Header.Get("Upload-Length"), 10, 64)
		if err != nil {
			http.Error(w, "Invalid Upload-Length header", http.StatusBadRequest)
			return
		}
		metadata := parseTusMetadata(r.Header.Get("Upload-Metadata"))
		filename := metadata["filename"]
		if filename == "" {
			http.Error(w, "No filename specified", http.StatusBadRequest)
			return
		}

		state := &uploadState{
			UserID: user.ID,
			Length: length,
		}

		file, err := h.Storage.File(r.Context(), user, path.Join("/", filename))
		if err != nil {
			logrus.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		state.File = file

		h.mutex.Lock()
		id := h.nextID
		h.nextID++
		h.uploads[id] = state
		h.mutex.Unlock()

		http.Redirect(w, r, fmt.Sprintf("/api/upload?resume=%d", id), http.StatusCreated)
		return
	}

	// if none of the tus protocol matches, we initiate a regular upload

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
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			filename := p.FileName()
			if filename == "" {
				http.Error(w, "Empty filename?", http.StatusBadRequest)
				return
			}

			f, err := h.Storage.File(r.Context(), user, path.Join("/", filename))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer f.Close()

			_, err = io.Copy(f, p)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusCreated)
		return
	}

	http.Error(w, "Invalid request, expected multipart", http.StatusBadRequest)
}

func parseTusMetadata(header string) map[string]string {
	meta := make(map[string]string)

	for _, element := range strings.Split(header, ",") {
		element := strings.TrimSpace(element)

		parts := strings.Split(element, " ")

		if len(parts) > 2 {
			continue
		}

		key := parts[0]
		if key == "" {
			continue
		}

		value := ""
		if len(parts) == 2 {
			// Ignore current element if the value is no valid base64
			dec, err := base64.StdEncoding.DecodeString(parts[1])
			if err != nil {
				continue
			}

			value = string(dec)
		}

		meta[key] = value
	}

	return meta
}

// the boolean returned will indicate whether we're done or not. if true that means we can be closed and removed from the book keeping
func (s *uploadState) Handle(w http.ResponseWriter, r *http.Request) bool {
	// A HEAD request simply wants to know where we left off, so the client knows from where to start
	if r.Method == http.MethodHead {
		w.Header().Add("Upload-Offset", fmt.Sprintf("%d", s.Position))
		w.Header().Add("Upload-Length", fmt.Sprintf("%d", s.Length))
		w.WriteHeader(http.StatusOK)
	} else if r.Method == http.MethodPatch {
		offset, err := strconv.ParseUint(r.Header.Get("Upload-Offset"), 10, 64)
		if err != nil {
			http.Error(w, "Invalid Upload-Offset header", http.StatusBadRequest)
			return false
		}
		if offset != s.Position {
			http.Error(w, fmt.Sprintf("Client and server are not at the same position, expected position %d, got %d", s.Position, offset), http.StatusInternalServerError)
			return false
		}

		// we copy the actual data to the end of our file
		n, err := io.Copy(s.File, r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return false
		}

		// then we calculate the next position by adding the amount of read bytes to our position
		s.Position += uint64(n)

		// and we report this new offset to the client
		w.Header().Add("Upload-Offset", fmt.Sprintf("%d", s.Position))
		w.WriteHeader(http.StatusNoContent)

		// if our new position is equal to the expected length we are done and can return true
		return s.Position == s.Length
	}

	return false
}

func (s *uploadState) Close() error {
	return s.File.Close()
}

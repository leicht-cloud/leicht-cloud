package webapi

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/leicht-cloud/leicht-cloud/pkg/app"
	"github.com/leicht-cloud/leicht-cloud/pkg/auth"
	"github.com/leicht-cloud/leicht-cloud/pkg/fileinfo"
	_ "github.com/leicht-cloud/leicht-cloud/pkg/fileinfo/builtin"
	"github.com/leicht-cloud/leicht-cloud/pkg/fileinfo/types"
	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
	"github.com/sirupsen/logrus"
)

type fileInfoHandler struct {
	Storage  storage.StorageProvider
	FileInfo *fileinfo.Manager
	Apps     *app.Manager
}

func newFileInfoHandler(store storage.StorageProvider, fileinfo *fileinfo.Manager, apps *app.Manager) http.Handler {
	return auth.AuthHandler(&fileInfoHandler{Storage: store, FileInfo: fileinfo, Apps: apps})
}

var websocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (h *fileInfoHandler) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")

	file, err := h.Storage.File(r.Context(), user, filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer file.Close()

	out, err := h.FileInfo.FileInfo(filename, file, &fileinfo.Options{Render: true})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	apps := h.Apps.Openers(out.MimeType)

	if websocket.IsWebSocketUpgrade(r) {
		conn, err := websocketUpgrader.Upgrade(w, r, nil)
		if err != nil {
			logrus.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = conn.WriteJSON(struct {
			Apps map[string]string `json:"apps"`
			Mime string            `json:"mime"`
		}{
			Apps: apps,
			Mime: out.MimeType.String(),
		})
		if err != nil {
			logrus.Error(err)
			conn.Close()
			return
		}

		go func(conn *websocket.Conn) {
			for {
				if _, _, err := conn.NextReader(); err != nil {
					logrus.Error(err)
					conn.Close()
					break
				}
			}
		}(conn)

		for info := range out.Channel {
			err = conn.WriteJSON(info)
			if err != nil {
				logrus.Error(err)
				conn.Close()
				return
			}
		}
	} else {
		outputs := make(map[string]types.Result)

		for info := range out.Channel {
			outputs[info.Name] = info
		}

		output := struct {
			Results map[string]types.Result `json:"data"`
			Apps    map[string]string       `json:"apps"`
			Mime    string                  `json:"mime"`
		}{
			Results: outputs,
			Apps:    apps,
			Mime:    out.MimeType.String(),
		}

		err = json.NewEncoder(w).Encode(output)
		if err != nil {
			logrus.Error(err)
		}
	}
}

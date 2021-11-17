package admin

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/schoentoon/go-cloud/pkg/auth"
	"github.com/schoentoon/go-cloud/pkg/plugin"
	"github.com/sirupsen/logrus"
)

type pluginStdoutHandler struct {
	PluginManager *plugin.Manager
}

var websocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (h *pluginStdoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil || !user.Admin {
		// we redirect you to / as you don't have permission to view the admin panel
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	name := r.URL.Query().Get("name")

	stdout, err := h.PluginManager.Stdout(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	go func(conn *websocket.Conn) {
		for {
			if _, _, err := conn.NextReader(); err != nil {
				conn.Close()
				stdout.Close()
				break
			}
		}
	}(conn)

	for line := range stdout.Channel() {
		err = conn.WriteMessage(websocket.TextMessage, line)
		if err != nil {
			logrus.Error(err)
			return
		}
	}
}

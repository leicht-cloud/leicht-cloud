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

	stdout := h.PluginManager.Stdout(name)
	logrus.Debug(stdout)

	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, line := range stdout {
		err = conn.WriteMessage(websocket.TextMessage, []byte(line))
		if err != nil {
			logrus.Error(err)
			conn.Close()
			return
		}
	}
}

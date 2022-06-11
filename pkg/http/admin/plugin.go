package admin

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/leicht-cloud/leicht-cloud/pkg/http/template"
	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/leicht-cloud/leicht-cloud/pkg/plugin"
	"github.com/sirupsen/logrus"
)

type pluginHandler struct {
	StaticHandler http.Handler
}

func (h *pluginHandler) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	// internal rewrite to admin page, so we render that
	r.URL.Path = "/admin.pluginview.gohtml"

	ctx := template.AttachTemplateData(r.Context(), struct {
		Name   string
		Navbar template.NavbarData
	}{
		Name: r.URL.Query().Get("name"),
		Navbar: template.NavbarData{
			Admin: user.Admin,
		},
	})

	h.StaticHandler.ServeHTTP(w, r.WithContext(ctx))
}

type pluginStdoutHandler struct {
	PluginManager *plugin.Manager
}

var websocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (h *pluginStdoutHandler) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
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

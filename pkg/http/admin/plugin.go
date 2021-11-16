package admin

import (
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/auth"
	"github.com/schoentoon/go-cloud/pkg/plugin"
)

type pluginStdoutHandler struct {
	PluginManager *plugin.Manager
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

	for _, line := range stdout {
		w.Write([]byte(line))
		w.Write([]byte{'\n'})
	}
}

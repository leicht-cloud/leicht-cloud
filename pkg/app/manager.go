package app

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/leicht-cloud/leicht-cloud/pkg/plugin"
	"github.com/leicht-cloud/leicht-cloud/pkg/prometheus"
)

type Manager struct {
	apps map[string]*App
}

func NewManager(pManager *plugin.Manager, prom *prometheus.Manager, apps ...string) (*Manager, error) {
	out := &Manager{
		apps: make(map[string]*App),
	}

	for _, name := range apps {
		plugin, err := pManager.Start(name, "app")
		if err != nil {
			return nil, err
		}

		app := &App{
			plugin: plugin,
		}

		out.apps[name] = app
	}

	return out, nil
}

func (m *Manager) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	split := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/apps/"), "/", 2)
	appname := split[0]
	path := "/"
	if len(split) == 2 {
		path = split[1]
	}

	app, ok := m.apps[appname]
	if !ok {
		http.Error(w, fmt.Sprintf("App not found: %s", appname), http.StatusNotFound)
		return
	}

	err := app.Serve(user, w, path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

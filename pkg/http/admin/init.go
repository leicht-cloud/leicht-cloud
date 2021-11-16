package admin

import (
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/plugin"
)

func Init(mux *http.ServeMux, templateHandler http.Handler, pluginManager *plugin.Manager) {
	mux.Handle("/admin/", &rootHandler{StaticHandler: templateHandler, PluginManager: pluginManager})
}

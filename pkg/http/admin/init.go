package admin

import (
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/plugin"
	"gorm.io/gorm"
)

func Init(mux *http.ServeMux, templateHandler http.Handler, pluginManager *plugin.Manager, db *gorm.DB) {
	mux.Handle("/admin/", &rootHandler{StaticHandler: templateHandler, PluginManager: pluginManager, DB: db})
	mux.Handle("/admin/plugin/stdout", &pluginStdoutHandler{PluginManager: pluginManager})
}

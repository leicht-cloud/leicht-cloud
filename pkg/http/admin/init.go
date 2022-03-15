package admin

import (
	"net/http"

	"github.com/leicht-cloud/leicht-cloud/pkg/auth"
	"github.com/leicht-cloud/leicht-cloud/pkg/plugin"
	"gorm.io/gorm"
)

func Init(mux *http.ServeMux, auth *auth.Provider, templateHandler http.Handler, pluginManager *plugin.Manager, db *gorm.DB) {
	mux.Handle("/admin/", Middleware(auth, &rootHandler{StaticHandler: templateHandler}))
	mux.Handle("/admin/userlist", Middleware(auth, &userlistHandler{StaticHandler: templateHandler, DB: db}))
	mux.Handle("/admin/user", Middleware(auth, &userHandler{StaticHandler: templateHandler, DB: db}))
	mux.Handle("/admin/plugin", Middleware(auth, &pluginHandler{StaticHandler: templateHandler}))
	mux.Handle("/admin/plugin/stdout", Middleware(auth, &pluginStdoutHandler{PluginManager: pluginManager}))
}

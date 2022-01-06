package admin

import (
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/auth"
	"github.com/schoentoon/go-cloud/pkg/http/template"
	"github.com/schoentoon/go-cloud/pkg/plugin"
	"gorm.io/gorm"
)

func Init(mux *http.ServeMux, auth *auth.Provider, templateHandler http.Handler, pluginManager *plugin.Manager, db *gorm.DB) {
	navbar := template.AdminNavbarData{
		Plugins: pluginManager.Plugins(),
	}

	mux.Handle("/admin/", Middleware(auth, &rootHandler{StaticHandler: templateHandler, AdminNavbar: navbar}))
	mux.Handle("/admin/userlist", Middleware(auth, &userlistHandler{StaticHandler: templateHandler, AdminNavbar: navbar, DB: db}))
	mux.Handle("/admin/user", Middleware(auth, &userHandler{StaticHandler: templateHandler, AdminNavbar: navbar, DB: db}))
	mux.Handle("/admin/plugin", Middleware(auth, &pluginHandler{AdminNavbar: navbar, StaticHandler: templateHandler}))
	mux.Handle("/admin/plugin/stdout", Middleware(auth, &pluginStdoutHandler{PluginManager: pluginManager}))
}

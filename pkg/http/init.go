package http

import (
	"net/http"

	"github.com/leicht-cloud/leicht-cloud/pkg/app"
	"github.com/leicht-cloud/leicht-cloud/pkg/auth"
	"github.com/leicht-cloud/leicht-cloud/pkg/fileinfo"
	"github.com/leicht-cloud/leicht-cloud/pkg/http/admin"
	"github.com/leicht-cloud/leicht-cloud/pkg/http/template"
	"github.com/leicht-cloud/leicht-cloud/pkg/http/webapi"
	"github.com/leicht-cloud/leicht-cloud/pkg/plugin"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
	"gorm.io/gorm"
)

func InitHttpServer(
	addr string,
	db *gorm.DB,
	authProvider *auth.Provider,
	storage storage.StorageProvider,
	pluginManager *plugin.Manager,
	apps *app.Manager,
	fileinfo *fileinfo.Manager,
) (*http.Server, error) {
	if addr == "" {
		addr = ":8080"
	}

	assets, err := InitStatic()
	if err != nil {
		return nil, err
	}
	templateHandler, err := template.NewHandler(assets, apps.Apps(), pluginManager.Plugins())
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("/", &rootHandler{DB: db, StaticHandler: templateHandler})
	mux.Handle("/login", &loginHandler{DB: db, Auth: authProvider, StaticHandler: templateHandler})
	mux.Handle("/signup", &signupHandler{Assets: assets, DB: db, Storage: storage})
	mux.Handle("/apps/embed/", auth.AuthHandler(apps))
	mux.Handle("/apps/", auth.AuthHandler(&appsHandler{Apps: apps, StaticHandler: templateHandler}))
	webapi.Init(mux, db, storage, fileinfo, apps)
	admin.Init(mux, authProvider, templateHandler, pluginManager, db)

	out := &http.Server{
		Addr: addr,
		Handler: auth.AuthMiddleware(authProvider,
			WithLogging(mux),
		),
	}
	return out, nil
}

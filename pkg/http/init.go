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
	db *gorm.DB,
	authProvider *auth.Provider,
	storage storage.StorageProvider,
	pluginManager *plugin.Manager,
	apps *app.Manager,
	fileinfo *fileinfo.Manager,
) (*http.Server, error) {
	assets, err := initStatic()
	if err != nil {
		return nil, err
	}
	templateHandler, err := template.NewHandler(assets)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("/", &rootHandler{DB: db, StaticHandler: templateHandler})
	mux.Handle("/login", &loginHandler{DB: db, Auth: authProvider, StaticHandler: templateHandler})
	mux.Handle("/signup", &signupHandler{Assets: assets, DB: db, Storage: storage})
	mux.Handle("/apps/", auth.AuthHandler(apps))
	webapi.Init(mux, db, storage, fileinfo)
	admin.Init(mux, authProvider, templateHandler, pluginManager, db)

	out := &http.Server{
		Addr: ":8080",
		Handler: auth.AuthMiddleware(authProvider,
			WithLogging(mux),
		),
	}
	return out, nil
}

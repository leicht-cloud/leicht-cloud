package http

import (
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/auth"
	"github.com/schoentoon/go-cloud/pkg/fileinfo"
	"github.com/schoentoon/go-cloud/pkg/http/admin"
	"github.com/schoentoon/go-cloud/pkg/http/api"
	"github.com/schoentoon/go-cloud/pkg/http/template"
	"github.com/schoentoon/go-cloud/pkg/plugin"
	"github.com/schoentoon/go-cloud/pkg/storage"
	"gorm.io/gorm"
)

func InitHttpServer(
	db *gorm.DB,
	authProvider *auth.Provider,
	storage storage.StorageProvider,
	pluginManager *plugin.Manager,
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
	api.Init(mux, db, storage, fileinfo)
	admin.Init(mux, templateHandler, pluginManager)

	out := &http.Server{
		Addr: ":8080",
		Handler: auth.AuthMiddleware(authProvider,
			WithLogging(mux),
		),
	}
	return out, nil
}

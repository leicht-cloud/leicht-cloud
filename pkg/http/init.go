package http

import (
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/auth"
	"github.com/schoentoon/go-cloud/pkg/http/api"
	"github.com/schoentoon/go-cloud/pkg/storage"
	"gorm.io/gorm"
)

func InitHttpServer(db *gorm.DB, authProvider *auth.Provider, storage storage.StorageProvider) (*http.Server, error) {
	assets, err := initStatic()
	if err != nil {
		return nil, err
	}
	templateHandler, err := NewTemplateHandler(assets)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("/", &rootHandler{DB: db, StaticHandler: templateHandler})
	mux.Handle("/login", &loginHandler{DB: db, Auth: authProvider, StaticHandler: templateHandler})
	mux.Handle("/signup", &signupHandler{Assets: assets, DB: db, Storage: storage})
	api.Init(mux, db, storage)

	out := &http.Server{
		Addr: ":8080",
		Handler: auth.AuthMiddleware(authProvider,
			WithLogging(mux),
		),
	}
	return out, nil
}

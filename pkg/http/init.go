package http

import (
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/auth"
	"github.com/schoentoon/go-cloud/pkg/http/api"
	"github.com/schoentoon/go-cloud/pkg/storage"
	"gorm.io/gorm"
)

func InitHttpServer(db *gorm.DB, auth *auth.Provider, storage storage.StorageProvider) (*http.Server, error) {
	assets, err := initStatic()
	if err != nil {
		return nil, err
	}
	staticHandler := http.FileServer(http.FS(assets))

	mux := http.NewServeMux()
	mux.Handle("/", &rootHandler{DB: db, Auth: auth, StaticHandler: staticHandler})
	mux.Handle("/login", &loginHandler{DB: db, Auth: auth, StaticHandler: staticHandler})
	mux.Handle("/signup", &signupHandler{Assets: assets, DB: db, Storage: storage})
	api.Init(mux, db, auth, storage)

	out := &http.Server{
		Addr:    ":8080",
		Handler: WithLogging(mux),
	}
	return out, nil
}

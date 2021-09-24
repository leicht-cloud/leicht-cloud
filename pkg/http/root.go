package http

import (
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/auth"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type rootHandler struct {
	DB            *gorm.DB
	Auth          *auth.Provider
	StaticHandler http.Handler
}

func (h *rootHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		h.StaticHandler.ServeHTTP(w, r)
		return
	}

	user, err := h.Auth.VerifyFromRequest(r)
	if err != nil {
		// internal we redirect you to signin.html
		r.URL.Path = "/signin.html"
		h.StaticHandler.ServeHTTP(w, r)
		return
	}

	log.Debug("Welcome ", user)
	// internal rewrite to the root folder
	r.URL.Path = "/folder.html"
	h.StaticHandler.ServeHTTP(w, r)
}

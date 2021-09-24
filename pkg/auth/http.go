package auth

import (
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/models"
)

func AuthHandler(authProvider *Provider, handler AuthHandlerInterface) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := authProvider.VerifyFromRequest(r)
		if err != nil {
			http.Error(w, "Invalid login", http.StatusForbidden)
			return
		}

		handler.Serve(user, w, r)
	})
}

type AuthHandlerInterface interface {
	Serve(user *models.User, w http.ResponseWriter, r *http.Request)
}

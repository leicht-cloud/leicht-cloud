package auth

import (
	"context"
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/models"
)

// AuthHandler you'll want to chain this in if you want to make auth mandatory for the endpoint
func AuthHandler(handler AuthHandlerInterface) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromRequest(r)
		if user == nil {
			http.Error(w, "Invalid login", http.StatusForbidden)
			return
		}

		handler.Serve(user, w, r)
	})
}

type AuthHandlerInterface interface {
	Serve(user *models.User, w http.ResponseWriter, r *http.Request)
}

const USER = "user"

// AuthMiddleware if auth should be optional and you want to do your own thing whenever the user is not logged in, use this
func AuthMiddleware(authProvider *Provider, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := authProvider.VerifyFromRequest(r)
		if err != nil {
			handler.ServeHTTP(w, r)
			return
		}

		handler.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), USER, user)))
	})
}

func GetUserFromRequest(r *http.Request) *models.User {
	user := r.Context().Value(USER)
	if user != nil {
		return user.(*models.User)
	}
	return nil
}

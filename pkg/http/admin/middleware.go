package admin

import (
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/auth"
	"github.com/schoentoon/go-cloud/pkg/models"
)

type middleWare struct {
	handler auth.AuthHandlerInterface
}

func Middleware(authProvider *auth.Provider, handler auth.AuthHandlerInterface) http.Handler {
	return auth.AuthHandler(&middleWare{
		handler: handler,
	})
}

func (m *middleWare) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	if user == nil || !user.Admin {
		// we redirect you to / as you don't have permission to view the admin panel
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	m.handler.Serve(user, w, r)
}

package http

import (
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/auth"
	"github.com/schoentoon/go-cloud/pkg/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type loginHandler struct {
	DB            *gorm.DB
	Auth          *auth.Provider
	StaticHandler http.Handler
}

func (h *loginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if auth.GetUserFromRequest(r) != nil {
		// you're already logged in, lol
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	if r.Method != http.MethodPost {
		// internal we redirect you to signin.html
		r.URL.Path = "/signin.html"
		h.StaticHandler.ServeHTTP(w, r)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !r.Form.Has("email") || !r.Form.Has("password") {
		http.Error(w, "Missing email or password, can't login.", http.StatusBadRequest)
		return
	}

	email := r.Form.Get("email")
	password := r.Form.Get("password")

	var user models.User

	result := h.DB.First(&user, "email = ?", email)

	if result.Error != nil {
		http.Error(w, "Wrong password", http.StatusForbidden)
		return
	}

	err = bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password))
	if err == nil {
		token, err := h.Auth.Authenticate(&user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:   "auth",
			Value:  token,
			MaxAge: 86400,
		})
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
	} else {
		http.Error(w, "Wrong password", http.StatusForbidden)
		return
	}
}

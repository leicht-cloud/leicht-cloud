package http

import (
	"io/fs"
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/storage"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type signupHandler struct {
	Assets  fs.FS
	DB      *gorm.DB
	Storage storage.StorageProvider
}

func (h *signupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		sendAsset(h.Assets, "signup.html", w, r)
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
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	user := models.User{
		Email:        email,
		PasswordHash: hash,
	}

	result := h.DB.Create(&user)

	if result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	} else {
		h.Storage.InitUser(r.Context(), &user) // TODO: implement actual error checking on this, ideally with a complete way of rolling back the previous progress
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
}

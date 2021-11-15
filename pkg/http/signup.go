package http

import (
	"io/fs"
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/storage"
	"github.com/sirupsen/logrus"
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

	// We create the user inside a transaction, so in case we fail to initialize something else
	// related to the new user we can easily undo the database part
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		tx = tx.Model(&models.User{}).Count(&count)
		if count == 0 {
			// TODO: For now we're making the first user admin by default, this can't remain like this obviously
			user.Admin = true
			logrus.Warn("First created user detected, so making it admin")
		}

		result := tx.Create(&user)

		if result.Error != nil {
			http.Error(w, result.Error.Error(), http.StatusInternalServerError)
			return result.Error
		} else {
			err = h.Storage.InitUser(r.Context(), &user)
			if err != nil {
				logrus.Errorf("Failed to initialize storage for new user, incorrect settings?: %s", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			}
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		}

		return nil
	})
	if err != nil {
		logrus.Error(err)
	}
}

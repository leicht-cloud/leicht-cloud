package admin

import (
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/http/template"
	"github.com/schoentoon/go-cloud/pkg/models"
	"gorm.io/gorm"
)

type userlistHandler struct {
	StaticHandler http.Handler
	AdminNavbar   template.AdminNavbarData
	DB            *gorm.DB
}

type userlistTemplateData struct {
	Navbar      template.NavbarData
	AdminNavbar template.AdminNavbarData

	Users []*models.User
}

func (h *userlistHandler) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	data := userlistTemplateData{
		Navbar: template.NavbarData{
			Admin: user.Admin,
		},
		AdminNavbar: h.AdminNavbar,
	}

	tx := h.DB.Find(&data.Users)

	if tx.Error != nil {
		http.Error(w, tx.Error.Error(), http.StatusInternalServerError)
		return
	}

	// internal rewrite to admin page, so we render that
	r.URL.Path = "/admin.userlist.gohtml"

	ctx := template.AttachTemplateData(r.Context(), data)

	h.StaticHandler.ServeHTTP(w, r.WithContext(ctx))
}

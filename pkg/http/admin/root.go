package admin

import (
	"net/http"

	"github.com/leicht-cloud/leicht-cloud/pkg/http/template"
	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"gorm.io/gorm"
)

type rootHandler struct {
	StaticHandler http.Handler
	AdminNavbar   template.AdminNavbarData
	DB            *gorm.DB
}

type adminTemplateData struct {
	Navbar      template.NavbarData
	AdminNavbar template.AdminNavbarData
}

func (h *rootHandler) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	// internal rewrite to admin page, so we render that
	r.URL.Path = "/admin.gohtml"

	data := adminTemplateData{
		Navbar: template.NavbarData{
			Admin: user.Admin,
		},
		AdminNavbar: h.AdminNavbar,
	}

	ctx := template.AttachTemplateData(r.Context(), data)

	h.StaticHandler.ServeHTTP(w, r.WithContext(ctx))
}

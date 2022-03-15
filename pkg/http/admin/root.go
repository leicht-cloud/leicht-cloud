package admin

import (
	"net/http"

	"github.com/leicht-cloud/leicht-cloud/pkg/http/template"
	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"gorm.io/gorm"
)

type rootHandler struct {
	StaticHandler http.Handler
	DB            *gorm.DB
}

type adminTemplateData struct {
	Navbar template.NavbarData
}

func (h *rootHandler) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	// internal rewrite to admin page, so we render that
	r.URL.Path = "/admin.gohtml"

	data := adminTemplateData{
		Navbar: template.NavbarData{
			Admin: user.Admin,
		},
	}

	ctx := template.AttachTemplateData(r.Context(), data)

	h.StaticHandler.ServeHTTP(w, r.WithContext(ctx))
}

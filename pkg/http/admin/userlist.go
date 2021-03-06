package admin

import (
	"net/http"

	"github.com/leicht-cloud/leicht-cloud/pkg/http/template"
	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"gorm.io/gorm"
)

type userlistHandler struct {
	StaticHandler http.Handler
	DB            *gorm.DB
}

type userlistTemplateData struct {
	Navbar template.NavbarData

	Users []*models.User
}

func (h *userlistHandler) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	data := userlistTemplateData{
		Navbar: template.NavbarData{
			Admin: user.Admin,
		},
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

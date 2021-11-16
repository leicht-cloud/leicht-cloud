package admin

import (
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/auth"
	"github.com/schoentoon/go-cloud/pkg/http/template"
	"github.com/schoentoon/go-cloud/pkg/plugin"
	"github.com/sirupsen/logrus"
)

type rootHandler struct {
	StaticHandler http.Handler
	PluginManager *plugin.Manager
}

type adminTemplateData struct {
	Navbar  template.NavbarData
	Plugins []string
}

func (h *rootHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil || !user.Admin {
		// we redirect you to / as you don't have permission to view the admin panel
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// internal rewrite to the root folder
	r.URL.Path = "/admin.gohtml"

	logrus.Debug(h.PluginManager.Plugins())

	ctx := template.AttachTemplateData(r.Context(), adminTemplateData{
		Navbar: template.NavbarData{
			Admin: user.Admin,
		},
		Plugins: h.PluginManager.Plugins(),
	})

	h.StaticHandler.ServeHTTP(w, r.WithContext(ctx))
}

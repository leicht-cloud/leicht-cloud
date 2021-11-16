package admin

import (
	"net/http"
	"strings"

	"github.com/schoentoon/go-cloud/pkg/auth"
	"github.com/schoentoon/go-cloud/pkg/http/template"
	"github.com/schoentoon/go-cloud/pkg/plugin"
)

type rootHandler struct {
	StaticHandler http.Handler
	PluginManager *plugin.Manager
}

type adminTemplateData struct {
	Navbar  template.NavbarData
	Plugins []string

	Page       string
	PluginView pluginView
}

type pluginView struct {
	Name string
}

func (h *rootHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil || !user.Admin {
		// we redirect you to / as you don't have permission to view the admin panel
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	data := adminTemplateData{
		Navbar: template.NavbarData{
			Admin: user.Admin,
		},
		Plugins: h.PluginManager.Plugins(),
	}

	if strings.HasPrefix(r.URL.Path, "/admin/plugin/") {
		data.Page = "plugin"
		data.PluginView = pluginView{
			Name: strings.TrimPrefix(r.URL.Path, "/admin/plugin/"),
		}
	}

	// internal rewrite to admin page, so we render that
	r.URL.Path = "/admin.gohtml"

	ctx := template.AttachTemplateData(r.Context(), data)

	h.StaticHandler.ServeHTTP(w, r.WithContext(ctx))
}

package http

import (
	"net/http"
	"strings"

	"github.com/leicht-cloud/leicht-cloud/pkg/app"
	"github.com/leicht-cloud/leicht-cloud/pkg/http/template"
	"github.com/leicht-cloud/leicht-cloud/pkg/models"
)

type appsHandler struct {
	Apps          *app.Manager
	StaticHandler http.Handler
}

type appTemplateData struct {
	Navbar      template.NavbarData
	App         string
	Permissions string
}

func (h *appsHandler) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	split := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/apps/"), "/", 2)
	appname := split[0]

	app, err := h.Apps.GetApp(appname)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// internal rewrite to the app page
	r.URL.Path = "/app.gohtml"

	ctx := template.AttachTemplateData(r.Context(), appTemplateData{
		Navbar: template.NavbarData{
			Admin: user.Admin,
		},
		App:         appname,
		Permissions: app.IFramePermissions(),
	})

	h.StaticHandler.ServeHTTP(w, r.WithContext(ctx))
}

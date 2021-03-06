package http

import (
	"net/http"
	"strings"

	"github.com/leicht-cloud/leicht-cloud/pkg/auth"
	"github.com/leicht-cloud/leicht-cloud/pkg/http/template"
	"gorm.io/gorm"
)

type rootHandler struct {
	DB            *gorm.DB
	StaticHandler http.Handler
}

type folderTemplateData struct {
	Navbar template.NavbarData
	Dir    []string
}

func (h *rootHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		h.StaticHandler.ServeHTTP(w, r)
		return
	}

	user := auth.GetUserFromRequest(r)
	if user == nil {
		// internal we redirect you to signin.html
		r.URL.Path = "/signin.html"
		h.StaticHandler.ServeHTTP(w, r)
		return
	}

	// internal rewrite to the root folder
	r.URL.Path = "/folder.gohtml"

	dir := r.URL.Query().Get("dir")
	if dir == "" {
		dir = "/"
	}

	dirSplit := []string{}

	for _, str := range strings.Split(dir, "/") {
		if str != "" {
			dirSplit = append(dirSplit, str)
		}
	}

	ctx := template.AttachTemplateData(r.Context(), folderTemplateData{
		Navbar: template.NavbarData{
			Admin: user.Admin,
		},
		Dir: dirSplit,
	})

	h.StaticHandler.ServeHTTP(w, r.WithContext(ctx))
}

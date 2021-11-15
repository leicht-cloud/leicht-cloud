package http

import (
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

type templateHandler struct {
	assets fs.FS

	template    *template.Template
	fileHandler http.Handler
}

func NewTemplateHandler(assets fs.FS) (*templateHandler, error) {
	// we first create an empty template so we can first attach the funcmap
	tmpl := template.New("")

	funcMap := template.FuncMap{
		"navbar": tmplFunc(assets, "includes/navbar.gohtml"),
	}
	tmpl.Funcs(funcMap)

	var err error
	tmpl, err = tmpl.ParseFS(assets, "*.gohtml")
	if err != nil {
		return nil, err
	}

	return &templateHandler{
		assets:      assets,
		template:    tmpl,
		fileHandler: http.FileServer(http.FS(assets)),
	}, nil
}

func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, ".gohtml") {
		err := t.template.ExecuteTemplate(w, strings.TrimLeft(r.URL.Path, "/"), nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		t.fileHandler.ServeHTTP(w, r)
	}
}

func tmplFunc(assets fs.FS, tmpl string) func() (template.HTML, error) {
	return func() (template.HTML, error) {
		f, err := assets.Open(tmpl)
		if err != nil {
			logrus.Error(err)
			return "", err
		}
		buf, err := io.ReadAll(f)
		if err != nil {
			logrus.Error(err)
			return "", err
		}
		return template.HTML(buf), nil
	}
}

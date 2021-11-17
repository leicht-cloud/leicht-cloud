//go:build html

package template

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

func NewHandler(assets fs.FS) (out *TemplateHandler, err error) {
	return &TemplateHandler{
		assets:      assets,
		template:    nil,
		fileHandler: http.FileServer(http.FS(assets)),
	}, nil
}

func getTemplate(assets fs.FS, name string) (*template.Template, error) {
	// we first create an empty template so we can first attach the funcmap
	tmpl := template.New("")

	funcMap, err := createFuncMap(assets)
	if err != nil {
		return nil, err
	}
	tmpl.Funcs(funcMap)

	tmpl, err = tmpl.ParseFS(assets, name)
	if err != nil {
		return nil, err
	}
	// we make sure there was just 1 template with this name
	for _, t := range tmpl.Templates() {
		if t.Name() == "" {
			continue
		} else if strings.HasSuffix(name, t.Name()) {
			return t, nil
		}
	}

	return nil, fmt.Errorf("Unable to find a template with the name %s", name)
}

func (t *TemplateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, ".gohtml") {
		tmpl, err := getTemplate(t.assets, strings.TrimLeft(r.URL.Path, "/"))
		if err != nil {
			logrus.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = tmpl.Execute(w, r.Context().Value(templateDataKeyValue))
		if err != nil {
			logrus.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		t.fileHandler.ServeHTTP(w, r)
	}
}

func tmplFunc(assets fs.FS, name string) func(data interface{}) (template.HTML, error) {
	// then we return a function with this specific template, resulting in it only being parsed once
	// TODO: Ideally we'd be writing directly to the main template somehow, rather than first writing into a string/buffer
	return func(data interface{}) (template.HTML, error) {
		tmpl, err := getTemplate(assets, name)
		if err != nil {
			return "", err
		}

		out := strings.Builder{}
		err = tmpl.Execute(&out, data)
		if err != nil {
			return "", err
		}
		return template.HTML(out.String()), nil
	}
}

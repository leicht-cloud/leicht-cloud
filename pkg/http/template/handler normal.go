//go:build !html

package template

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
)

func NewHandler(assets fs.FS, apps, plugins []string) (out *TemplateHandler, err error) {
	// we first create an empty template so we can first attach the funcmap
	tmpl := template.New("")

	funcMap, err := createFuncMap(assets, apps, plugins)
	if err != nil {
		return nil, err
	}
	tmpl.Funcs(funcMap)

	tmpl, err = tmpl.ParseFS(assets, "*.gohtml")
	if err != nil {
		return nil, err
	}

	return &TemplateHandler{
		assets:      assets,
		template:    tmpl,
		fileHandler: http.FileServer(http.FS(assets)),
		apps:        apps,
		plugins:     plugins,
	}, nil
}

func (t *TemplateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, ".gohtml") {
		err := t.template.ExecuteTemplate(w, strings.TrimLeft(r.URL.Path, "/"), r.Context().Value(templateDataKeyValue))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		t.fileHandler.ServeHTTP(w, r)
	}
}

func tmplFunc(assets fs.FS, name string, funcMap template.FuncMap) func(data interface{}) (template.HTML, error) {
	tmpl := template.New("")

	// we attach the funcMap
	tmpl = tmpl.Funcs(funcMap)

	tmpl, err := tmpl.ParseFS(assets, name)
	if err != nil {
		panic(err)
	}

	// we should have 2 templates now, the first empty one that we created just so we could attach a function map
	// and the second one that we actually want..
	if len(tmpl.Templates()) != 2 {
		panic(fmt.Errorf("Found an odd amount of templates with the name %s in assets", name))
	}

	// then we change the main template to the one we were out for
	tmpl = tmpl.Templates()[1]

	// then we return a function with this specific template, resulting in it only being parsed once
	// TODO: Ideally we'd be writing directly to the main template somehow, rather than first writing into a string/buffer
	return func(data interface{}) (template.HTML, error) {
		out := strings.Builder{}
		err := tmpl.Execute(&out, data)
		if err != nil {
			return "", err
		}
		return template.HTML(out.String()), nil
	}
}

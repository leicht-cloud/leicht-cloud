package template

import (
	"context"
	"html/template"
	"io/fs"
	"net/http"
)

type TemplateHandler struct {
	assets fs.FS

	template    *template.Template
	fileHandler http.Handler
}

func createFuncMap(assets fs.FS) (out template.FuncMap, err error) {
	// we gracefully handle panics in here, as tmplFunc may panic when it fails
	// to load a template. this solution is cleaner than making it return an error
	// as I can still just put the calls directly into the FuncMap initialization
	// rather than having to handle errors for every single one manually
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			}
		}
	}()

	return template.FuncMap{
		"navbar":      tmplFunc(assets, "includes/navbar.gohtml"),
		"adminnavbar": tmplFunc(assets, "includes/navbar.admin.gohtml"),
		"notnil": func(data interface{}) bool {
			return data != nil
		},
	}, err
}

type templateDataKey int

var templateDataKeyValue templateDataKey

func AttachTemplateData(ctx context.Context, data interface{}) context.Context {
	return context.WithValue(ctx, templateDataKeyValue, data)
}

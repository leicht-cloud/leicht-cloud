//go:build !html

package http

import (
	"embed"
	"io/fs"
)

//go:embed assets
var assets embed.FS

func InitStatic() (fs.FS, error) {
	embedded, err := fs.Sub(assets, "assets")
	if err != nil {
		return nil, err
	}
	return embedded, nil
}

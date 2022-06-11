//go:build html

package http

import (
	"io/fs"
	"os"
)

func InitStatic() (fs.FS, error) {
	return os.DirFS("assets"), nil
}

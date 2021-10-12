//go:build html

package http

import (
	"io/fs"
	"os"
)

func initStatic() (fs.FS, error) {
	return os.DirFS("assets"), nil
}

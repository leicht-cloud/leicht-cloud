package http

import (
	"io"
	"io/fs"
	"log"
	"net/http"
)

func sendAsset(assets fs.FS, filename string, w http.ResponseWriter, r *http.Request) {
	asset, err := assets.Open(filename)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	defer asset.Close()

	_, err = io.Copy(w, asset)
	if err != nil {
		log.Println(err)
	}
}

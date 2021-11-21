package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/vp8l"
	_ "golang.org/x/image/webp"
)

type ImageSize struct {
}

type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (is *ImageSize) MinimumBytes(typ, subtyp string) (int64, error) {
	if typ != "image" {
		return 0, errors.New("Not an image")
	}

	// TODO: 1024 is an assumption here, ideally check with most of the more popular image formats
	return 1024, nil
}

func (is *ImageSize) Check(filename string, reader io.Reader) ([]byte, error) {
	cfg, _, err := image.DecodeConfig(reader)
	if err != nil {
		return nil, err
	}
	out := Size{Width: cfg.Width, Height: cfg.Height}
	return json.Marshal(out)
}

func (is *ImageSize) Render(data []byte) (string, error) {
	var cfg Size
	err := json.Unmarshal(data, &cfg)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("height: %d, width: %d", cfg.Height, cfg.Width), nil
}

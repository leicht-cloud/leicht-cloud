package types

import (
	"errors"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
)

var ErrSkip = errors.New("Skip")

var providers = make(map[string]FileInfoProvider)
var mimeprovider = make(map[string]MimeTypeProvider)

func RegisterProvider(name string, provider FileInfoProvider) {
	providers[name] = provider
}

func RegisterMimeProvider(name string, provider MimeTypeProvider) {
	mimeprovider[name] = provider
}

func GetProvider(name string) (FileInfoProvider, error) {
	p, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("No provider found with the name: %s", name)
	}
	return p, nil
}

func GetMimeProvider(name string) (MimeTypeProvider, error) {
	// as a default we try to load libmagic first, if that doesn't work
	// we fall back to the gonative implementation.
	if name == "" {
		out, err := getMimeProvider("libmagic")
		if err == nil {
			logrus.Info("No mime provider specified, using libmagic")
			return out, nil
		}
		logrus.Infof("No mime provider specified, tried using libmagic but failed with: %s", err)
		return getMimeProvider("gonative")
	}
	return getMimeProvider(name)
}

func getMimeProvider(name string) (MimeTypeProvider, error) {
	p, ok := mimeprovider[name]
	if !ok {
		return nil, fmt.Errorf("No provider found with the name: %s", name)
	}
	return p, p.Init()
}

type FileInfoProvider interface {
	// indicate how many bytes you need at minimum to be able to determine
	// your specific file info. you won't get more than provided. in case you
	// need all the data, please return -1. In case the provided mime type doesn't
	// suit you and you know you won't be able to do anything with the data, returning
	// any error from here will skip the provider
	MinimumBytes(typ, subtyp string) (int64, error)

	Check(filename string, reader io.Reader) ([]byte, error)
	Render(data []byte) (string, string, error)
}

type MimeTypeProvider interface {
	Init() error
	io.Closer

	MinimumBytes() int64

	MimeType(filename string, reader io.Reader) (*MimeType, error)
}

type Result struct {
	Name  string `json:"name"`
	Data  []byte `json:"data"`
	Human string `json:"human"`
	Title string `json:"title"`
	Err   error  `json:"error,omitempty"`
}

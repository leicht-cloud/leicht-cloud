package types

import (
	"errors"
	"fmt"
	"io"
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
	p, ok := mimeprovider[name]
	if !ok {
		return nil, fmt.Errorf("No provider found with the name: %s", name)
	}
	return p, nil
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
	MinimumBytes() int64

	MimeType(filename string, reader io.Reader) (*MimeType, error)
}

type MimeType struct {
	Type    string `json:"type"`
	SubType string `json:"subtype"`
}

func (m MimeType) String() string {
	return fmt.Sprintf("%s/%s", m.Type, m.SubType)
}

type Result struct {
	Name  string `json:"name"`
	Data  []byte `json:"data"`
	Human string `json:"human"`
	Title string `json:"title"`
	Err   error  `json:"error,omitempty"`
}

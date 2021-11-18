package fileinfo

import (
	"fmt"
	"io"
)

var providers map[string]FileInfoProvider

func RegisterProvider(name string, provider FileInfoProvider) {
	providers[name] = provider
}

func GetProvider(name string) (FileInfoProvider, error) {
	p, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("No provider found with the name: %s", name)
	}
	return p, nil
}

type FileInfoProvider interface {
	// indicate how many bytes you need at minimum to be able to determine
	// your specific file info. you won't get more than provided. in case you
	// need all the data, please return -1
	MinimumBytes() int64

	Check(filename string, reader io.Reader) (interface{}, error)
}

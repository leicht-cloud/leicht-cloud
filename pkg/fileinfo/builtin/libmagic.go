//go:build cgo
// +build cgo

package builtin

import (
	"io"
	"strings"

	fileinfo "github.com/leicht-cloud/leicht-cloud/pkg/fileinfo/types"
	"github.com/wenerme/go-magic"
)

func init() {
	fileinfo.RegisterMimeProvider("libmagic", &libmagicMimeTypeProvider{})
}

type libmagicMimeTypeProvider struct {
	libmagic magic.Magic
}

func (m *libmagicMimeTypeProvider) Init() error {
	m.libmagic = magic.Open(magic.MAGIC_NONE)

	err := m.libmagic.Load("")
	if err != nil {
		return err
	}

	m.libmagic.SetFlags(magic.MAGIC_MIME_TYPE)

	return nil
}

func (m *libmagicMimeTypeProvider) Close() error {
	if m.libmagic != 0 {
		return m.libmagic.Close()
	}
	return nil
}

func (m *libmagicMimeTypeProvider) MinimumBytes() int64 {
	// on first glance there doesn't seem to be a set minimum bytes for libmagic
	// however when simply running strace on the `file` command (which uses libmagic)
	// I saw it reading 32KB.. so we'll go with that, even though lower is probably perfectly fine
	//
	// upon inspecting the source of the `file` command it turns out it tries to read 1024 * 1024
	// https://github.com/file/file/blob/0eb7c1b83341cc954620b45d2e2d65ee7df1a4e7/src/file.h#L484
	// seeing how 32KB seems to work just fine, we'll keep it at that
	return 1024 * 32
}

func (m *libmagicMimeTypeProvider) MimeType(filename string, reader io.Reader) (*fileinfo.MimeType, error) {
	buf, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	raw := m.libmagic.Buffer(buf)
	split := strings.SplitN(raw, "/", 2)

	return &fileinfo.MimeType{
		Type:    split[0],
		SubType: split[1],
	}, nil
}

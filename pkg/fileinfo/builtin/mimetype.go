package builtin

import (
	"io"

	"github.com/h2non/filetype"
	"github.com/h2non/filetype/types"
	fileinfo "github.com/schoentoon/go-cloud/pkg/fileinfo/types"
)

func init() {
	fileinfo.RegisterMimeProvider("gonative", &GoNativeMimeTypeProvider{})
}

type GoNativeMimeTypeProvider struct {
}

func (m *GoNativeMimeTypeProvider) MinimumBytes() int64 {
	// according to the README of the filetype library we would only need 262, however
	// issues point out this isn't always the case https://github.com/h2non/filetype/issues/107
	// also ideally this will get changed to a constant from their library the moment
	// https://github.com/h2non/filetype/issues/83 is addressed
	return 262
}

func (m *GoNativeMimeTypeProvider) MimeType(filename string, reader io.Reader) (*fileinfo.MimeType, error) {
	typ, err := filetype.MatchReader(reader)
	if err != nil {
		return nil, err
	}

	// we explicitly rewrite the unknown type to application/octet-stream
	if typ == types.Unknown {
		return &fileinfo.MimeType{
			Type:    "application",
			SubType: "octet-stream",
		}, nil
	}

	return &fileinfo.MimeType{
		Type:    typ.MIME.Type,
		SubType: typ.MIME.Subtype,
	}, nil
}

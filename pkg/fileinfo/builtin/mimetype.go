package builtin

import (
	"io"

	"github.com/h2non/filetype"
	"github.com/schoentoon/go-cloud/pkg/fileinfo"
)

func init() {
	fileinfo.RegisterProvider("mimetype", &MimeTypeProvider{})
}

type MimeTypeProvider struct {
}

func (m *MimeTypeProvider) MinimumBytes() int64 {
	// according to the README of the filetype library we would only need 262, however
	// issues point out this isn't always the case https://github.com/h2non/filetype/issues/107
	// also ideally this will get changed to a constant from their library the moment
	// https://github.com/h2non/filetype/issues/83 is addressed
	return 262
}

func (m *MimeTypeProvider) Check(filename string, reader io.Reader) (interface{}, error) {
	typ, err := filetype.MatchReader(reader)
	if err != nil {
		return nil, err
	}
	return typ, nil
}

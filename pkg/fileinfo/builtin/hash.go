package builtin

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"

	"github.com/schoentoon/go-cloud/pkg/fileinfo"
)

func init() {
	fileinfo.RegisterProvider("md5", newHashProvider(func() hash.Hash { return md5.New() }))
	fileinfo.RegisterProvider("sha1", newHashProvider(func() hash.Hash { return sha1.New() }))
	fileinfo.RegisterProvider("sha256", newHashProvider(func() hash.Hash { return sha256.New() }))
	fileinfo.RegisterProvider("sha512", newHashProvider(func() hash.Hash { return sha512.New() }))
}

type hashProvider struct {
	hasher func() hash.Hash
}

func newHashProvider(fn func() hash.Hash) *hashProvider {
	return &hashProvider{
		hasher: fn,
	}
}

func (h *hashProvider) MinimumBytes() int64 {
	return -1
}

func (h *hashProvider) Check(filename string, reader io.Reader) (interface{}, error) {
	out := h.hasher()
	_, err := io.Copy(out, reader)
	if err != nil {
		return nil, err
	}

	return out.Sum(nil), nil
}

func (h *hashProvider) Render(data interface{}) string {
	raw := data.([]byte)

	return fmt.Sprintf("%x", raw)
}

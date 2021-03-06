package builtin

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"

	"golang.org/x/crypto/sha3"

	fileinfo "github.com/leicht-cloud/leicht-cloud/pkg/fileinfo/types"
)

func init() {
	fileinfo.RegisterProvider("md5", newHashProvider("MD5", func() hash.Hash { return md5.New() }))
	fileinfo.RegisterProvider("sha1", newHashProvider("SHA1", func() hash.Hash { return sha1.New() }))
	fileinfo.RegisterProvider("sha256", newHashProvider("SHA256", func() hash.Hash { return sha256.New() }))
	fileinfo.RegisterProvider("sha384", newHashProvider("SHA384", func() hash.Hash { return sha512.New384() }))
	fileinfo.RegisterProvider("sha512", newHashProvider("SHA512", func() hash.Hash { return sha512.New() }))
	fileinfo.RegisterProvider("sha3-224", newHashProvider("SHA3-224", func() hash.Hash { return sha3.New224() }))
	fileinfo.RegisterProvider("sha3-256", newHashProvider("SHA3-256", func() hash.Hash { return sha3.New256() }))
	fileinfo.RegisterProvider("sha3-384", newHashProvider("SHA3-384", func() hash.Hash { return sha3.New384() }))
	fileinfo.RegisterProvider("sha3-512", newHashProvider("SHA3-512", func() hash.Hash { return sha3.New512() }))
}

type hashProvider struct {
	hasher func() hash.Hash
	title  string
}

func newHashProvider(title string, fn func() hash.Hash) *hashProvider {
	return &hashProvider{
		hasher: fn,
		title:  title,
	}
}

func (h *hashProvider) MinimumBytes(typ, subtyp string) (int64, error) {
	return -1, nil
}

func (h *hashProvider) Check(filename string, reader io.Reader) ([]byte, error) {
	out := h.hasher()
	_, err := io.Copy(out, reader)
	if err != nil {
		return nil, err
	}

	return out.Sum(nil), nil
}

func (h *hashProvider) Render(data []byte) (string, string, error) {
	return fmt.Sprintf("%x", data), h.title, nil
}

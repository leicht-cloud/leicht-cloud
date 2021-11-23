package fileinfo

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/schoentoon/go-cloud/pkg/fileinfo/types"
	"github.com/stretchr/testify/assert"
)

func RunFileInfoProviderFromFile(fileinfo types.FileInfoProvider, filename string, tb testing.TB, expect []byte) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		tb.Skip(err)
	}

	RunFileInfoProvider(fileinfo, filename, data, tb, expect)
}

func RunFileInfoProvider(fileinfo types.FileInfoProvider, filename string, data []byte, tb testing.TB, expect []byte) {
	reader := bytes.NewReader(data) // we create the reader first, so it doesn't count as an alloc per operation

	switch tb := tb.(type) {
	case *testing.T:
		out, err := fileinfo.Check(filename, reader)
		if assert.NoError(tb, err) {
			assert.Equal(tb, expect, out)
		}
	case *testing.B:
		tb.ResetTimer()

		for i := 0; i < tb.N; i++ {
			reader.Reset(data)
			out, err := fileinfo.Check(filename, reader)
			if assert.NoError(tb, err) {
				assert.Equal(tb, expect, out)
			}
		}
	}
}

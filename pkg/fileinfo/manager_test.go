package fileinfo

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	_ "github.com/leicht-cloud/leicht-cloud/pkg/fileinfo/builtin"
	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage/builtin/memory"
	"github.com/stretchr/testify/assert"
)

var promManager, _ = (&prometheus.Config{Enabled: false}).Create()

func newTestFS(t *testing.T) *memory.StorageProvider {
	return memory.NewStorageProvider()
}

func fill(t *testing.T, store *memory.StorageProvider, filename string, size int64) {
	data, err := ioutil.ReadAll(io.LimitReader(rand.Reader, size))
	if err != nil {
		t.Skip(err)
	}
	store.Data[filename] = data
}

type testProvider struct {
	t       *testing.T
	min     int64
	scanMax bool
	output  []byte
}

func (t *testProvider) MinimumBytes(typ, subtyp string) (int64, error) {
	assert.Equal(t.t, "application", typ)
	assert.Equal(t.t, "octet-stream", subtyp)
	return t.min, nil
}

func (t *testProvider) Check(filename string, reader io.Reader) ([]byte, error) {
	if t.scanMax {
		n, err := io.CopyN(io.Discard, reader, t.min)
		assert.NoError(t.t, err)
		assert.Equal(t.t, t.min, n)
	} else {
		n, err := io.Copy(io.Discard, reader)
		assert.NoError(t.t, err)
		assert.GreaterOrEqual(t.t, n, t.min)
	}
	return t.output, nil
}

func (t *testProvider) Render(d []byte) (string, string, error) {
	return fmt.Sprintf("render: %s", d), "title", nil
}

type panicProvider struct {
}

func (p *panicProvider) MinimumBytes(typ, subtyp string) (int64, error) {
	return 1, nil
}

func (p *panicProvider) Check(filename string, reader io.Reader) ([]byte, error) {
	panic(errors.New("panic, lol"))
}

func (p *panicProvider) Render([]byte) (string, string, error) {
	return "", "", nil
}

func TestLimitedRead(t *testing.T) {
	store := newTestFS(t)
	fill(t, store, "/1024", 1024)

	manager, err := NewManager(nil, promManager, "gonative")
	assert.NoError(t, err)
	manager.providers["test"] = &testProvider{
		t:   t,
		min: 512,
	}

	file, err := store.File(context.Background(), &models.User{}, "/1024")
	if assert.NoError(t, err) {
		_, err := manager.FileInfo("/1024", file, &Options{})
		assert.NoError(t, err)
	}
}

func TestMultiprocess(t *testing.T) {
	store := newTestFS(t)
	fill(t, store, "/1024", 1024)

	manager, err := NewManager(nil, promManager, "gonative")
	assert.NoError(t, err)
	manager.providers["test"] = &testProvider{
		t:      t,
		min:    512,
		output: []byte{9, 0, 0, 1},
	}
	manager.providers["test2"] = &testProvider{
		t:      t,
		min:    1024,
		output: []byte("Test output"),
	}
	manager.providers["test3"] = &testProvider{
		t:       t,
		min:     512,
		scanMax: true,
	}

	file, err := store.File(context.Background(), &models.User{}, "/1024")
	if assert.NoError(t, err) {
		out, err := manager.FileInfo("/1024", file, &Options{Render: true}, "test", "test2", "test3")
		assert.NoError(t, err)

		count := 0

		for info := range out.Channel {
			assert.NoError(t, info.Err)
			assert.NotEmpty(t, info.Human)
			count++
			switch info.Name {
			case "mime":
				continue
			case "test":
				assert.Equal(t, []byte{9, 0, 0, 1}, info.Data)
			case "test2":
				assert.Equal(t, []byte("Test output"), info.Data)
			case "test3":
				assert.Equal(t, []byte(nil), info.Data)
			default:
				assert.Failf(t, "Unexpected info", "Unexpected info structure %#v", info)
			}
		}

		assert.Equal(t, 4, count)
	}
}

func TestPanic(t *testing.T) {
	store := newTestFS(t)
	fill(t, store, "/1024", 1024)

	manager, err := NewManager(nil, promManager, "gonative")
	assert.NoError(t, err)
	manager.providers["panic"] = &panicProvider{}

	file, err := store.File(context.Background(), &models.User{}, "/1024")
	if assert.NoError(t, err) {
		data, err := manager.FileInfo("/1024", file, &Options{}, "panic")
		assert.NoError(t, err)

		count := 0

		for info := range data.Channel {
			count++
			if info.Name == "mime" { // we don't really care to check the mime structure
				continue
			}
			assert.Error(t, info.Err)
			assert.Equal(t, "panic", info.Name)
		}
		assert.Equal(t, 2, count)
	}
}

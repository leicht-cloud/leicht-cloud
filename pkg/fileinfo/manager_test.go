package fileinfo

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"io/ioutil"
	"testing"

	_ "github.com/schoentoon/go-cloud/pkg/fileinfo/builtin"
	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/storage/builtin/memory"
	"github.com/stretchr/testify/assert"
)

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

func (t *testProvider) Render([]byte) (string, error) {
	return "", nil
}

type panicProvider struct {
}

func (p *panicProvider) MinimumBytes(typ, subtyp string) (int64, error) {
	return 1, nil
}

func (p *panicProvider) Check(filename string, reader io.Reader) ([]byte, error) {
	panic(errors.New("panic, lol"))
}

func (p *panicProvider) Render([]byte) (string, error) {
	return "", nil
}

func TestLimitedRead(t *testing.T) {
	store := newTestFS(t)
	fill(t, store, "/1024", 1024)

	manager, err := NewManager(nil, "gonative")
	assert.NoError(t, err)
	manager.providers["test"] = &testProvider{
		t:   t,
		min: 512,
	}

	file, err := store.File(context.Background(), &models.User{}, "/1024")
	if assert.NoError(t, err) {
		_, err := manager.FileInfo("/1024", file, &Options{}, "test")
		assert.NoError(t, err)
	}
}

func TestMultiprocess(t *testing.T) {
	store := newTestFS(t)
	fill(t, store, "/1024", 1024)

	manager, err := NewManager(nil, "gonative")
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
		out, err := manager.FileInfo("/1024", file, &Options{}, "test", "test2", "test3")
		assert.NoError(t, err)

		assert.Len(t, out.Data, 3)
		assert.Equal(t, []byte{9, 0, 0, 1}, out.Data["test"].Data)
		assert.NoError(t, out.Data["test"].Err)
		assert.Equal(t, []byte("Test output"), out.Data["test2"].Data)
		assert.NoError(t, out.Data["test2"].Err)
		assert.Equal(t, []byte(nil), out.Data["test3"].Data)
		assert.NoError(t, out.Data["test3"].Err)
	}
}

func TestPanic(t *testing.T) {
	store := newTestFS(t)
	fill(t, store, "/1024", 1024)

	manager, err := NewManager(nil, "gonative")
	assert.NoError(t, err)
	manager.providers["panic"] = &panicProvider{}

	file, err := store.File(context.Background(), &models.User{}, "/1024")
	if assert.NoError(t, err) {
		data, err := manager.FileInfo("/1024", file, &Options{}, "panic")
		assert.NoError(t, err)
		assert.Error(t, data.Data["panic"].Err)
	}
}

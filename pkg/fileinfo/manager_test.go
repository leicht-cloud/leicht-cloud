package fileinfo

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"io/ioutil"
	"testing"

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
	output  interface{}
}

func (t *testProvider) MinimumBytes() int64 {
	return t.min
}

func (t *testProvider) Check(filename string, reader io.Reader) (interface{}, error) {
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

type panicProvider struct {
}

func (p *panicProvider) MinimumBytes() int64 {
	return 1
}

func (p *panicProvider) Check(filename string, readerr io.Reader) (interface{}, error) {
	panic(errors.New("panic, lol"))
}

func TestLimitedRead(t *testing.T) {
	store := newTestFS(t)
	fill(t, store, "/1024", 1024)

	manager, err := NewManager()
	assert.NoError(t, err)
	manager.providers["test"] = &testProvider{
		t:   t,
		min: 512,
	}

	file, err := store.File(context.Background(), &models.User{}, "/1024")
	if assert.NoError(t, err) {
		_, err := manager.FileInfo("/1024", file, "test")
		assert.NoError(t, err)
	}
}

func TestMultiprocess(t *testing.T) {
	store := newTestFS(t)
	fill(t, store, "/1024", 1024)

	manager, err := NewManager()
	assert.NoError(t, err)
	manager.providers["test"] = &testProvider{
		t:      t,
		min:    512,
		output: 9001,
	}
	manager.providers["test2"] = &testProvider{
		t:      t,
		min:    1024,
		output: "Test output",
	}
	manager.providers["test3"] = &testProvider{
		t:       t,
		min:     512,
		scanMax: true,
	}

	file, err := store.File(context.Background(), &models.User{}, "/1024")
	if assert.NoError(t, err) {
		out, err := manager.FileInfo("/1024", file, "test", "test2", "test3")
		assert.NoError(t, err)

		assert.Len(t, out, 3)
		assert.Equal(t, out["test"], 9001)
		assert.Equal(t, out["test2"], "Test output")
		assert.Equal(t, out["test3"], nil)
	}
}

func TestPanic(t *testing.T) {
	store := newTestFS(t)
	fill(t, store, "/1024", 1024)

	manager, err := NewManager()
	assert.NoError(t, err)
	manager.providers["panic"] = &panicProvider{}

	file, err := store.File(context.Background(), &models.User{}, "/1024")
	if assert.NoError(t, err) {
		_, err := manager.FileInfo("/1024", file, "panic")
		assert.Error(t, err)
	}
}

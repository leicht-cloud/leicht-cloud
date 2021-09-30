package storage

// This is really only exposed so you have an easy way to test your own storage provider
// All you really have to add to your tests is the following
//
// func TestLocal(t *testing.T) {
// 	provider := &StorageProvider{RootPath: t.TempDir()} // init your own provider here
//
// 	storage.TestStorageProvider(provider, t)
// }

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/schoentoon/go-cloud/pkg/models"
)

// TODO: Add benchmarks and fuzzing?

func TestStorageProvider(provider StorageProvider, t *testing.T) {
	user := &models.User{
		ID:    1337,
		Email: "test@test.com",
	}

	t.Run("InitUser", func(t *testing.T) { testInitUser(t, user, provider) })
	t.Run("Mkdir", func(t *testing.T) { testMkdir(t, user, provider) })
	t.Run("ListDirectory", func(t *testing.T) { testListDirectory(t, user, provider) })
	if t.Run("File/1KB", func(t *testing.T) { testFile(t, user, provider, 1024) }) {
		// we only continue with the large file tests if the first one actually passed.
		t.Run("File/4KB", func(t *testing.T) { testFile(t, user, provider, 1024*4) })
		t.Run("File/8KB", func(t *testing.T) { testFile(t, user, provider, 1024*4) })
		//t.Run("File/1MB", func(t *testing.T) { testFile(t, user, provider, 1024*1024) })
		//t.Run("File/8MB", func(t *testing.T) { testFile(t, user, provider, 1024*1024*8) })
		//t.Run("File/16MB", func(t *testing.T) { testFile(t, user, provider, 1024*1024*16) })
	}
	t.Run("Delete", func(t *testing.T) { testDelete(t, user, provider) })
}

func testInitUser(t *testing.T, user *models.User, storage StorageProvider) {
	assert.NoError(t, storage.InitUser(context.Background(), user))
}

func testMkdir(t *testing.T, user *models.User, storage StorageProvider) {
	assert.NoError(t, storage.Mkdir(context.Background(), user, "random/dir"))
}

func testListDirectory(t *testing.T, user *models.User, storage StorageProvider) {
	dir, err := storage.ListDirectory(context.Background(), user, "random")
	assert.NoError(t, err)

	assert.Len(t, dir.Files, 1)
	assert.Equal(t, "dir", dir.Files[0].Name)
}

func testFile(t *testing.T, user *models.User, storage StorageProvider, size int) {
	buffer := make([]byte, size)
	n, err := rand.Read(buffer)
	assert.NoError(t, err)
	assert.Equal(t, size, n)

	filename := fmt.Sprintf("file-%d", size)

	if !t.Run("Write", func(t *testing.T) {
		file, err := storage.File(context.Background(), user, filename)
		if !assert.NoError(t, err) {
			return
		}

		n, err := file.Write(buffer)
		assert.NoError(t, err)
		assert.Equal(t, size, n)

		assert.NoError(t, file.Close())
	}) {
		return
	}

	// This for some reasons seems needed for the file based ones?
	// tests are executed too quickly??
	syscall.Sync()

	if !t.Run("List", func(t *testing.T) {
		dir, err := storage.ListDirectory(context.Background(), user, ".")
		if !assert.NoError(t, err) {
			return
		}

		found := false
		for _, entry := range dir.Files {
			if entry.Name == filename {
				found = true
				assert.Equal(t, uint64(size), entry.Size)
			}
		}

		assert.True(t, found)
	}) {
		return
	}

	if !t.Run("Read", func(t *testing.T) {
		file, err := storage.File(context.Background(), user, filename)
		assert.NoError(t, err)

		data, err := io.ReadAll(file)
		if assert.NoError(t, err) {
			assert.Equal(t, buffer, data)
		}
	}) {
		return
	}

	if !t.Run("Delete", func(t *testing.T) {
		err := storage.Delete(context.Background(), user, filename)
		assert.NoError(t, err)

		// File should ALWAYS return a file struct
		file, err := storage.File(context.Background(), user, filename)
		assert.NoError(t, err)

		// however the first read call should return an error
		_, err = io.ReadAll(file)
		assert.Error(t, err)
	}) {
		return
	}
}

func testDelete(t *testing.T, user *models.User, storage StorageProvider) {
	err := storage.Delete(context.Background(), user, "random/dir")
	assert.NoError(t, err)

	dir, err := storage.ListDirectory(context.Background(), user, "random")
	assert.NoError(t, err)

	assert.Len(t, dir.Files, 0)
}

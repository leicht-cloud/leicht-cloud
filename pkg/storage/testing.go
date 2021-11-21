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
	if t.Run("File/1KB", func(t *testing.T) { testFile(t, user, provider, 1024) }) {
		// we only continue with the large file tests if the first one actually passed.
		t.Run("File/4KB", func(t *testing.T) { testFile(t, user, provider, 1024*4) })
		t.Run("File/8KB", func(t *testing.T) { testFile(t, user, provider, 1024*8) })
		t.Run("File/16KB", func(t *testing.T) { testFile(t, user, provider, 1024*16) })
		t.Run("File/32KB", func(t *testing.T) { testFile(t, user, provider, 1024*32) })
		t.Run("File/1MB", func(t *testing.T) { testFile(t, user, provider, 1024*1024) })
		//t.Run("File/8MB", func(t *testing.T) { testFile(t, user, provider, 1024*1024*8) })
		//t.Run("File/16MB", func(t *testing.T) { testFile(t, user, provider, 1024*1024*16) })
	}
}

func testInitUser(t *testing.T, user *models.User, storage StorageProvider) {
	assert.NoError(t, storage.InitUser(context.Background(), user))
}

func testMkdir(t *testing.T, user *models.User, storage StorageProvider) {
	assert.NoError(t, storage.Mkdir(context.Background(), user, "random/dir"))
}

func testFile(t *testing.T, user *models.User, storage StorageProvider, size int) {
	buffer := make([]byte, size)
	n, err := rand.Read(buffer)
	assert.NoError(t, err)
	assert.Equal(t, size, n)

	filename := fmt.Sprintf("file-%d", size)
	moved := fmt.Sprintf("moved-%d", size)

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

	if !t.Run("ListPreMove", func(t *testing.T) {
		dir, err := storage.ListDirectory(context.Background(), user, "/")
		if !assert.NoError(t, err) {
			return
		}

		found := false
		for file := range dir {
			if file.Name == filename {
				found = true
				assert.Equal(t, uint64(size), file.Size)
			}
		}
		assert.True(t, found)
	}) {
		return
	}

	if !t.Run("Move", func(t *testing.T) {
		assert.NoError(t, storage.Move(context.Background(), user, filename, moved))
	}) {
		return
	}

	if !t.Run("ListPostMove", func(t *testing.T) {
		dir, err := storage.ListDirectory(context.Background(), user, "/")
		if !assert.NoError(t, err) {
			return
		}

		found := false
		for file := range dir {
			if file.Name == moved {
				found = true
				assert.Equal(t, uint64(size), file.Size)
			}
		}
		assert.True(t, found)
	}) {
		return
	}

	if !t.Run("Read", func(t *testing.T) {
		file, err := storage.File(context.Background(), user, moved)
		assert.NoError(t, err)

		data, err := io.ReadAll(file)
		if assert.NoError(t, err) {
			assert.Equal(t, buffer, data)
		}

		assert.NoError(t, file.Close())
	}) {
		return
	}

	if !t.Run("Delete", func(t *testing.T) {
		err := storage.Delete(context.Background(), user, moved)
		assert.NoError(t, err)

		// File should ALWAYS return a file struct
		file, err := storage.File(context.Background(), user, moved)
		assert.NoError(t, err)

		// however the first read call should return an error
		_, err = io.ReadAll(file)
		assert.Error(t, err)
	}) {
		return
	}
}

func BenchmarkStorageProvider(storage StorageProvider, b *testing.B) {
	user := &models.User{
		ID:    1337,
		Email: "test@test.com",
	}

	if b.Run("File/1KB", func(b *testing.B) { benchmarkFile(b, user, storage, 1024) }) {
		b.Run("File/4KB", func(b *testing.B) { benchmarkFile(b, user, storage, 1024*4) })
		b.Run("File/8KB", func(b *testing.B) { benchmarkFile(b, user, storage, 1024*8) })
		b.Run("File/16KB", func(b *testing.B) { benchmarkFile(b, user, storage, 1024*16) })
		b.Run("File/32KB", func(b *testing.B) { benchmarkFile(b, user, storage, 1024*32) })
		b.Run("File/1MB", func(b *testing.B) { benchmarkFile(b, user, storage, 1024*1024) })
	}
}

func benchmarkFile(b *testing.B, user *models.User, provider StorageProvider, size int) {
	filename := fmt.Sprintf("file-%d", size)
	defer func(b *testing.B) {
		assert.NoError(b, provider.Delete(context.Background(), user, filename))
	}(b)

	assert.NoError(b, provider.InitUser(context.Background(), user))

	if !b.Run("Write", func(b *testing.B) {
		buffer := make([]byte, size)
		n, err := rand.Read(buffer)
		assert.NoError(b, err)
		assert.Equal(b, size, n)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			file, err := provider.File(context.Background(), user, filename)
			if !assert.NoError(b, err) {
				return
			}

			n, err := file.Write(buffer)
			assert.NoError(b, err)
			assert.Equal(b, size, n)
			b.SetBytes(int64(n))

			assert.NoError(b, file.Close())
		}
	}) {
		return
	}

	if !b.Run("Read", func(b *testing.B) {
		buf := make([]byte, size)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			file, err := provider.File(context.Background(), user, filename)
			if !assert.NoError(b, err) {
				return
			}

			n, err := io.CopyBuffer(io.Discard, file, buf)
			if err == io.EOF {
				break
			}
			assert.NoError(b, err)
			b.SetBytes(int64(n))
			assert.NoError(b, file.Close())
		}
		b.StopTimer()
	}) {
		return
	}
}

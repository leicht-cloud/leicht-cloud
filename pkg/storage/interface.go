package storage

import (
	"context"
	"io"
	"time"

	"github.com/schoentoon/go-cloud/pkg/models"
)

type FileInfo struct {
	Name      string    `json:"name"`
	FullPath  string    `json:"full_path"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Size      uint64    `json:"size"`
	Directory bool      `json:"directory"`
}

type File interface {
	io.ReadCloser
	io.WriteCloser
}

// TODO: Add support for https://pkg.go.dev/io#ReaderAt and https://pkg.go.dev/io#WriterAt ?
type StorageProvider interface {
	InitUser(ctx context.Context, user *models.User) error
	Mkdir(ctx context.Context, user *models.User, path string) error
	Move(ctx context.Context, user *models.User, src, dst string) error
	ListDirectory(ctx context.Context, user *models.User, path string) (<-chan FileInfo, error)
	File(ctx context.Context, user *models.User, fullpath string) (File, error)
	Delete(ctx context.Context, user *models.User, fullpath string) error
}

// Implement this interface if you want to be notified after your config is loaded, in case
// you need to do additional initialization using the config values
type PostConfigure interface {
	OnConfigure() error
}

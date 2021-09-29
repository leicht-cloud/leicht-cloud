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
	MimeType  string    `json:"mime_type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Size      uint64    `json:"size"`
}

type DirectoryInfo struct {
	Path  string     `json:"path"`
	Files []FileInfo `json:"files"`
}

type File interface {
	io.ReadCloser
	io.WriteCloser
}

type StorageProvider interface {
	InitUser(ctx context.Context, user *models.User) error
	Mkdir(ctx context.Context, user *models.User, path string) error
	Move(ctx context.Context, user *models.User, src, dst string) error
	ListDirectory(ctx context.Context, user *models.User, path string) (*DirectoryInfo, error)
	File(ctx context.Context, user *models.User, fullpath string) (File, error)
	Delete(ctx context.Context, user *models.User, fullpath string) error
}

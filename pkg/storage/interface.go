package storage

import (
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
	InitUser(user *models.User) error
	Mkdir(user *models.User, path string) error
	Move(user *models.User, src, dst string) error
	ListDirectory(user *models.User, path string) (*DirectoryInfo, error)
	File(user *models.User, fullpath string) (File, error)
}

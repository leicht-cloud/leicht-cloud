package local

import (
	"context"
	"os"
	"path"

	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/storage"
)

type StorageProvider struct {
	RootPath string
}

func (s *StorageProvider) InitUser(ctx context.Context, user *models.User) error {
	return os.MkdirAll(path.Join(s.RootPath, user.Email), 0700)
}

func (s *StorageProvider) Mkdir(ctx context.Context, user *models.User, dir string) error {
	return os.MkdirAll(path.Join(s.RootPath, user.Email, dir), 0700)
}

func (s *StorageProvider) Move(ctx context.Context, user *models.User, src, dst string) error {
	return os.Rename(path.Join(s.RootPath, user.Email, src), path.Join(s.RootPath, user.Email, dst))
}

func (s *StorageProvider) ListDirectory(ctx context.Context, user *models.User, dir string) (*storage.DirectoryInfo, error) {
	direntires, err := os.ReadDir(path.Join(s.RootPath, user.Email, dir))
	if err != nil {
		return nil, err
	}

	out := &storage.DirectoryInfo{
		Path:  dir,
		Files: make([]storage.FileInfo, 0, len(direntires)),
	}

	for _, entry := range direntires {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		out.Files = append(out.Files, storage.FileInfo{
			Name:      entry.Name(),
			FullPath:  path.Join(dir, entry.Name()),
			MimeType:  "",
			CreatedAt: info.ModTime(),
			UpdatedAt: info.ModTime(),
			Size:      uint64(info.Size()),
		})
	}

	return out, nil
}

func (s *StorageProvider) File(ctx context.Context, user *models.User, fullpath string) (storage.File, error) {
	return &File{
		FullPath: path.Join(s.RootPath, user.Email, fullpath),
	}, nil
}

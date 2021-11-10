package memory

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/storage"
)

// This provider should NOT be used in production and also has no concept of users
type StorageProvider struct {
	Data map[string][]byte
}

func NewStorageProvider() *StorageProvider {
	return &StorageProvider{
		Data: make(map[string][]byte),
	}
}

func (s *StorageProvider) joinPath(user *models.User, dir string) string {
	return path.Join("/", dir)
}

func (s *StorageProvider) InitUser(ctx context.Context, user *models.User) error {
	return nil
}

func (s *StorageProvider) Mkdir(ctx context.Context, user *models.User, dir string) error {
	return nil
}

func (s *StorageProvider) Move(ctx context.Context, user *models.User, src, dst string) error {
	srcFile, ok := s.Data[s.joinPath(user, src)]
	if !ok {
		return fmt.Errorf("Couldn't find src file with the name: %s", src)
	}
	s.Data[s.joinPath(user, dst)] = srcFile
	delete(s.Data, s.joinPath(user, src))
	return nil
}

func (s *StorageProvider) ListDirectory(ctx context.Context, user *models.User, dir string) (<-chan storage.FileInfo, error) {
	out := make(chan storage.FileInfo)

	go func(out chan<- storage.FileInfo, dir string) {
		for key, file := range s.Data {
			if strings.HasPrefix(key, dir) {
				out <- storage.FileInfo{
					Name:     strings.TrimPrefix(key, dir),
					FullPath: key,
					Size:     uint64(len(file)),
				}
			}
		}

		close(out)
	}(out, dir)

	return out, nil
}

func (s *StorageProvider) File(ctx context.Context, user *models.User, fullpath string) (storage.File, error) {
	srcFile, ok := s.Data[s.joinPath(user, fullpath)]
	if !ok {
		return &File{
			Data:     make([]byte, 0),
			filename: s.joinPath(user, fullpath),
			provider: s,
		}, nil
	}
	return &File{
		Data:     srcFile,
		filename: s.joinPath(user, fullpath),
		provider: s,
	}, nil
}

func (s *StorageProvider) Delete(ctx context.Context, user *models.User, fullpath string) error {
	delete(s.Data, s.joinPath(user, fullpath))
	return nil
}

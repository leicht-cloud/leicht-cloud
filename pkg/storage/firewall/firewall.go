package firewall

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage/utils"
)

var ErrReadOnly = errors.New("Readonly storage")

type FirewallStorageProvider struct {
	proxy     storage.StorageProvider
	directory string
}

func Firewall(provider storage.StorageProvider, directory string) storage.StorageProvider {
	return &FirewallStorageProvider{proxy: provider, directory: directory}
}

func (f *FirewallStorageProvider) InitUser(ctx context.Context, user *models.User) error {
	return f.proxy.InitUser(ctx, user)
}

func (f *FirewallStorageProvider) Mkdir(ctx context.Context, user *models.User, path string) error {
	err := utils.ValidatePath(path)
	if err != nil {
		return err
	}

	return f.proxy.Mkdir(ctx, user, filepath.Join(f.directory, path))
}

func (f *FirewallStorageProvider) Move(ctx context.Context, user *models.User, src string, dst string) error {
	err := utils.ValidatePath(src)
	if err != nil {
		return err
	}
	err = utils.ValidatePath(dst)
	if err != nil {
		return err
	}

	return f.proxy.Move(ctx, user, filepath.Join(f.directory, src), filepath.Join(f.directory, dst))
}

func (f *FirewallStorageProvider) ListDirectory(ctx context.Context, user *models.User, path string) (<-chan storage.FileInfo, error) {
	err := utils.ValidatePath(path)
	if err != nil {
		return nil, err
	}

	return f.proxy.ListDirectory(ctx, user, filepath.Join(f.directory, path))
}

func (f *FirewallStorageProvider) File(ctx context.Context, user *models.User, fullpath string) (storage.File, error) {
	err := utils.ValidatePath(fullpath)
	if err != nil {
		return nil, err
	}

	return f.proxy.File(ctx, user, filepath.Join(f.directory, fullpath))
}

func (f *FirewallStorageProvider) Delete(ctx context.Context, user *models.User, fullpath string) error {
	err := utils.ValidatePath(fullpath)
	if err != nil {
		return err
	}

	return f.proxy.Delete(ctx, user, filepath.Join(f.directory, fullpath))
}

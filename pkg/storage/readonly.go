package storage

import (
	"context"
	"errors"

	"github.com/leicht-cloud/leicht-cloud/pkg/models"
)

var ErrReadOnly = errors.New("Readonly storage")

type ReadonlyStorage struct {
	proxy StorageProvider
}

func ReadOnly(provider StorageProvider) StorageProvider {
	return &ReadonlyStorage{proxy: provider}
}

func (r *ReadonlyStorage) InitUser(ctx context.Context, user *models.User) error {
	return r.proxy.InitUser(ctx, user)
}

func (r *ReadonlyStorage) Mkdir(ctx context.Context, user *models.User, path string) error {
	return ErrReadOnly
}

func (r *ReadonlyStorage) Move(ctx context.Context, user *models.User, src string, dst string) error {
	return ErrReadOnly
}

func (r *ReadonlyStorage) ListDirectory(ctx context.Context, user *models.User, path string) (<-chan FileInfo, error) {
	return r.proxy.ListDirectory(ctx, user, path)
}

func (r *ReadonlyStorage) File(ctx context.Context, user *models.User, fullpath string) (File, error) {
	file, err := r.proxy.File(ctx, user, fullpath)
	if err != nil {
		return nil, err
	}

	return &readOnlyFile{proxy: file}, nil
}

func (r *ReadonlyStorage) Delete(ctx context.Context, user *models.User, fullpath string) error {
	return ErrReadOnly
}

type readOnlyFile struct {
	proxy File
}

func (r *readOnlyFile) Read(p []byte) (n int, err error) {
	return r.proxy.Read(p)
}

func (r *readOnlyFile) Close() error {
	return r.proxy.Close()
}

func (r *readOnlyFile) Write(p []byte) (n int, err error) {
	return -1, ErrReadOnly
}

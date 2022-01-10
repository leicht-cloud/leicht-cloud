package utils

import (
	"context"
	"errors"
	"path/filepath"
	"unicode"

	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
)

var ErrDirectoryBack = errors.New("Attempted to go a directory back")
var ErrInvisibleCharacter = errors.New("Invisible character in input")

func ValidatePath(path string) error {
	split := filepath.SplitList(path)
	for _, part := range split {
		err := validatePart(part)
		if err != nil {
			return err
		}
	}
	return nil
}

func validatePart(part string) error {
	if part == ".." {
		return ErrDirectoryBack
	}

	for _, char := range part {
		if !unicode.IsPrint(char) {
			return ErrInvisibleCharacter
		}
	}

	return nil
}

// Wrap around an existing StorageProvider and call ValidatePath on all the path arguments

type ValidateWrapper struct {
	Next storage.StorageProvider
}

func (w *ValidateWrapper) InitUser(ctx context.Context, user *models.User) error {
	return w.Next.InitUser(ctx, user)
}

func (w *ValidateWrapper) Mkdir(ctx context.Context, user *models.User, path string) error {
	err := ValidatePath(path)
	if err != nil {
		return err
	}
	return w.Next.Mkdir(ctx, user, path)
}

func (w *ValidateWrapper) Move(ctx context.Context, user *models.User, src string, dst string) error {
	err := ValidatePath(src)
	if err != nil {
		return err
	}
	err = ValidatePath(dst)
	if err != nil {
		return err
	}
	return w.Next.Move(ctx, user, src, dst)
}

func (w *ValidateWrapper) ListDirectory(ctx context.Context, user *models.User, path string) (<-chan storage.FileInfo, error) {
	err := ValidatePath(path)
	if err != nil {
		return nil, err
	}
	return w.Next.ListDirectory(ctx, user, path)
}

func (w *ValidateWrapper) File(ctx context.Context, user *models.User, fullpath string) (storage.File, error) {
	err := ValidatePath(fullpath)
	if err != nil {
		return nil, err
	}
	return w.Next.File(ctx, user, fullpath)
}

func (w *ValidateWrapper) Delete(ctx context.Context, user *models.User, fullpath string) error {
	err := ValidatePath(fullpath)
	if err != nil {
		return err
	}
	return w.Next.Delete(ctx, user, fullpath)
}

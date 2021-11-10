package local

import (
	"testing"

	"github.com/schoentoon/go-cloud/pkg/storage"
)

func TestLocal(t *testing.T) {
	provider := &StorageProvider{RootPath: t.TempDir()}

	storage.TestStorageProvider(provider, t)
}

package local

import (
	"testing"

	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
)

func TestLocal(t *testing.T) {
	provider := NewStorageProvider(t.TempDir())

	storage.TestStorageProvider(provider, t)
}

func BenchmarkLocal(b *testing.B) {
	provider := NewStorageProvider(b.TempDir())

	storage.BenchmarkStorageProvider(provider, b)
}

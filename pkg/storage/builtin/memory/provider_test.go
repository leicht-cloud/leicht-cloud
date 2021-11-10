package memory

import (
	"testing"

	"github.com/schoentoon/go-cloud/pkg/storage"
)

func TestMemory(t *testing.T) {
	provider := NewStorageProvider()

	storage.TestStorageProvider(provider, t)
}

func BenchmarkMemory(b *testing.B) {
	provider := NewStorageProvider()

	storage.BenchmarkStorageProvider(provider, b)
}

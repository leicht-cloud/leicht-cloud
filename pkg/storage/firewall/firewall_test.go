package firewall

import (
	"context"
	"testing"

	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage/builtin/local"
	"github.com/stretchr/testify/assert"
)

func TestFirewallStorage(t *testing.T) {
	contents := []byte("This is just a simple test.")
	user := &models.User{
		ID:    1337,
		Email: "test@test.com",
	}

	store := local.NewStorageProvider(t.TempDir())
	provider := Firewall(store, "test/directory")

	storage.TestStorageProvider(provider, t)

	if !t.Run("WriteInFirewall", func(t *testing.T) {
		file, err := provider.File(context.Background(), user, "test.txt")
		if assert.Nil(t, err) {
			_, err = file.Write(contents)
			assert.Nil(t, err)
		}
	}) {
		return
	}

	t.Run("ReadOutsideFirewall", func(t *testing.T) {
		file, err := store.File(context.Background(), user, "test/directory/test.txt")
		if assert.Nil(t, err) {
			buf := make([]byte, 1024)
			n, err := file.Read(buf)
			assert.Nil(t, err)

			assert.Equal(t, contents, buf[:n])
		}
	})
}

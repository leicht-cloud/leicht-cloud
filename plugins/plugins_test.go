package plugin

import (
	"fmt"
	"os"
	"testing"

	"github.com/schoentoon/go-cloud/pkg/plugin"
	"github.com/schoentoon/go-cloud/pkg/storage"
	storagePlugin "github.com/schoentoon/go-cloud/pkg/storage/plugin"
	"github.com/stretchr/testify/assert"
)

func TestPlugins(t *testing.T) {
	dir, err := os.ReadDir(".")
	assert.NoError(t, err)

	pluginManager := plugin.NewManager(".")
	defer pluginManager.Close()

	//testPlugin(t, pluginManager, "local-storage")

	for _, entry := range dir {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(fmt.Sprintf("%s/%s", entry.Name(), entry.Name())); os.IsNotExist(err) {
			t.Logf("%s isn't compiled?", entry.Name())
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) { testPlugin(t, pluginManager, entry.Name()) })
	}
}

func testPlugin(t *testing.T, pManager *plugin.Manager, name string) {
	conn, err := pManager.Start(name)
	if err != nil {
		t.Skip(err)
		return
	}

	// TODO: Read from a file in the plugin directory
	cfg := map[interface{}]interface{}{
		"path": t.TempDir(),
	}

	provider, err := storagePlugin.NewGrpcStorage(conn, cfg)
	assert.NoError(t, err)

	storage.TestStorageProvider(provider, t)
}

package plugin

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/leicht-cloud/leicht-cloud/pkg/plugin"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
	storagePlugin "github.com/leicht-cloud/leicht-cloud/pkg/storage/plugin"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestPlugins(t *testing.T) {
	runPlugins(t)
}

func BenchmarkPlugins(b *testing.B) {
	runPlugins(b)
}

func runPlugins(tb testing.TB) {
	dir, err := os.ReadDir(".")
	assert.NoError(tb, err)

	cfg := plugin.Config{
		Path:       []string{"."},
		WorkDir:    tb.TempDir(),
		Namespaced: new(bool),
	}
	*cfg.Namespaced = true

	// inside CI we will NOT run namespaced, as it means we're probably running inside docker
	if os.Getenv("CI") != "" {
		*cfg.Namespaced = false
	}

	pluginManager, err := cfg.CreateManager()
	assert.NoError(tb, err)
	defer pluginManager.Close()

	for _, entry := range dir {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(fmt.Sprintf("%s/%s", entry.Name(), entry.Name())); os.IsNotExist(err) {
			tb.Logf("%s isn't compiled?", entry.Name())
			continue
		}

		manifest, err := plugin.ParseManifestFromFile(entry.Name())
		if assert.NoError(tb, err) {
			// if we're not of type storage, we skip
			if manifest.Type != "storage" {
				continue
			}
		} else {
			continue
		}

		switch tb := tb.(type) {
		case *testing.T:
			tb.Run(entry.Name(), func(t *testing.T) { testPlugin(t, pluginManager, entry.Name()) })
		case *testing.B:
			tb.Run(entry.Name(), func(b *testing.B) { benchmarkPlugin(b, pluginManager, entry.Name()) })
		}
	}
}

func testPlugin(t *testing.T, pManager *plugin.Manager, name string) {
	store, err := setupTestEnv(pManager, name, t.TempDir())
	if err != nil {
		t.Skip(err)
	}
	defer func() {
		err = teardownTestEnv(name)
		assert.NoError(t, err)
	}()

	assert.NoError(t, err)

	storage.TestStorageProvider(store, t)
}

func benchmarkPlugin(b *testing.B, pManager *plugin.Manager, name string) {
	store, err := setupTestEnv(pManager, name, b.TempDir())
	if err != nil {
		b.Skip(err)
	}
	defer func() {
		err = teardownTestEnv(name)
		assert.NoError(b, err)
	}()

	assert.NoError(b, err)

	storage.BenchmarkStorageProvider(store, b)
}

func setupTestEnv(pManager *plugin.Manager, name, tmpdir string) (storage.StorageProvider, error) {
	conn, err := pManager.Start(name)
	if err != nil {
		return nil, err
	}

	cfg := map[interface{}]interface{}{
		"path": tmpdir,
	}

	// if the plugin directory has a config.test.yml file we load this and pass it along
	info, err := os.Stat(fmt.Sprintf("%s/config.test.yml", name))
	if err == nil && info.Mode().IsRegular() {
		f, err := os.Open(fmt.Sprintf("%s/config.test.yml", name))
		if err != nil {
			return nil, err
		}
		decoder := yaml.NewDecoder(f)
		err = decoder.Decode(&cfg)
		if err != nil {
			return nil, err
		}
	}

	// if there's a pre-test.sh script we execute this beforehand, from here you could
	// start a docker container with the service you're communicating with or something alike
	info, err = os.Stat(fmt.Sprintf("%s/pre-test.sh", name))
	if err == nil && info.Mode().IsRegular() {
		cmd := exec.Command(fmt.Sprintf("%s/pre-test.sh", name))
		buf := new(bytes.Buffer)
		// we do add a TMPDIR env variable for you to use as a temporary directory
		// do however keep in mind that golang while clean this out automatically
		// and the test could still fail if it can't do this. in the case of docker
		// run the container as the current user, you can do this by adding `-u "$(id -u):$(id -g)"`
		// to your docker run command
		cmd.Env = append(os.Environ(), fmt.Sprintf("TMPDIR=%s", tmpdir))
		cmd.Stdout = buf
		err = cmd.Run()
		if err != nil {
			return nil, err
		}
	}

	return storagePlugin.NewGrpcStorage(conn, cfg)
}

func teardownTestEnv(name string) error {
	// we will also want to run post-test.sh if it's there, to clean up stuff we setup in pre-test.sh
	info, err := os.Stat(fmt.Sprintf("%s/post-test.sh", name))
	if err == nil && info.Mode().IsRegular() {
		cmd := exec.Command(fmt.Sprintf("%s/post-test.sh", name))
		return cmd.Run()
	}
	return nil
}

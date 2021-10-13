package plugin

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/schoentoon/go-cloud/pkg/plugin"
	"github.com/schoentoon/go-cloud/pkg/storage"
	storagePlugin "github.com/schoentoon/go-cloud/pkg/storage/plugin"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestPlugins(t *testing.T) {
	dir, err := os.ReadDir(".")
	assert.NoError(t, err)

	pluginManager := plugin.NewManager(".")
	defer pluginManager.Close()

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

	// if the plugin directory has a config.test.yml file we load this and pass it along
	info, err := os.Stat(fmt.Sprintf("%s/config.test.yml", name))
	if err == nil && info.Mode().IsRegular() {
		f, err := os.Open(fmt.Sprintf("%s/config.test.yml", name))
		if err != nil {
			t.Fatal(err)
		}
		decoder := yaml.NewDecoder(f)
		err = decoder.Decode(&cfg)
		if err != nil {
			t.Fatal(err)
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
		cmd.Env = append(os.Environ(), fmt.Sprintf("TMPDIR=%s", t.TempDir()))
		cmd.Stdout = buf
		err = cmd.Run()
		if err != nil {
			t.Skipf("Skipping %s test: %s %s", name, buf.String(), err)
		}
	}

	provider, err := storagePlugin.NewGrpcStorage(conn, cfg)
	assert.NoError(t, err)

	storage.TestStorageProvider(provider, t)

	// we will also want to run post-test.sh if it's there, to clean up stuff we setup in pre-test.sh
	info, err = os.Stat(fmt.Sprintf("%s/post-test.sh", name))
	if err == nil && info.Mode().IsRegular() {
		cmd := exec.Command(fmt.Sprintf("%s/post-test.sh", name))
		err = cmd.Run()
		if err != nil {
			t.Logf("Error while cleaning up: %s", err)
		}
	}
}

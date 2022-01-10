package storage

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"gopkg.in/yaml.v2"
)

func SetupTestEnv(provider StorageProvider, tmpdir string) (StorageProvider, error) {
	// if the plugin directory has a config.test.yml file we load this and pass it along
	info, err := os.Stat("./config.test.yml")
	if err == nil && info.Mode().IsRegular() {
		f, err := os.Open("./config.test.yml")
		if err != nil {
			return nil, err
		}
		decoder := yaml.NewDecoder(f)
		err = decoder.Decode(provider)
		if err != nil {
			return nil, err
		}
	}

	if onconfig, ok := provider.(PostConfigure); ok {
		err = onconfig.OnConfigure()
		if err != nil {
			return nil, err
		}
	}

	// if there's a pre-test.sh script we execute this beforehand, from here you could
	// start a docker container with the service you're communicating with or something alike
	info, err = os.Stat("./pre-test.sh")
	if err == nil && info.Mode().IsRegular() {
		cmd := exec.Command("./pre-test.sh")
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

	return provider, nil
}

func TeardownTestEnv() error {
	// we will also want to run post-test.sh if it's there, to clean up stuff we setup in pre-test.sh
	info, err := os.Stat("./post-test.sh")
	if err == nil && info.Mode().IsRegular() {
		cmd := exec.Command("./post-test.sh")
		return cmd.Run()
	}
	return nil
}

package plugin

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var (
	ErrNoName = errors.New("No name specified")
)

type Manifest struct {
	Name        string      `yaml:"name"`
	Type        string      `yaml:"type"`
	Permissions Permissions `yaml:"permissions"`
	Prometheus  bool        `yaml:"prometheus"`
}

type Permissions struct {
	Container struct {
		Network bool `yaml:"network"`
	} `yaml:"container"`
	App struct {
		Javascript bool `yaml:"javascript"`
		Storage    struct {
			Enabled    bool `yaml:"enabled"`
			ReadWrite  bool `yaml:"readwrite"`
			WholeStore bool `yaml:"wholestore"`
		} `yaml:"storage"`
	} `yaml:"app"`
}

// path should be the path to the plugin, not directly to the manifest
func ParseManifestFromFile(path, typ string) (*Manifest, error) {
	f, err := os.Open(filepath.Join(path, "plugin.manifest.yml"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseManifest(f, typ)
}

func parseManifest(r io.Reader, typ string) (*Manifest, error) {
	out := &Manifest{}
	err := yaml.NewDecoder(r).Decode(out)
	if err != nil {
		return nil, err
	}
	if out.Name == "" {
		return nil, ErrNoName
	}
	if typ != "" && out.Type != typ {
		return nil, fmt.Errorf("Unwanted type: %s", out.Type)
	}
	return out, nil
}

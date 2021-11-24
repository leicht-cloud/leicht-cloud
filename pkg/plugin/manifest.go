package plugin

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// the valid plugin types
var types = []string{"storage", "fileinfo"}

var (
	ErrNoName = errors.New("No name specified")
)

type Manifest struct {
	Name        string      `yaml:"name"`
	Type        string      `yaml:"type"`
	Permissions Permissions `yaml:"permissions"`
}

type Permissions struct {
	Network bool `yaml:"network"`
}

// path should be the path to the plugin, not directly to the manifest
func ParseManifestFromFile(path string) (*Manifest, error) {
	f, err := os.Open(filepath.Join(path, "plugin.manifest.yml"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseManifest(f)
}

func parseManifest(r io.Reader) (*Manifest, error) {
	out := &Manifest{}
	err := yaml.NewDecoder(r).Decode(out)
	if err != nil {
		return nil, err
	}
	if out.Name == "" {
		return nil, ErrNoName
	}
	if !validType(out.Type) {
		return nil, fmt.Errorf("Invalid type: %s", out.Type)
	}
	return out, nil
}

func validType(typ string) bool {
	for _, t := range types {
		if t == typ {
			return true
		}
	}
	return false
}

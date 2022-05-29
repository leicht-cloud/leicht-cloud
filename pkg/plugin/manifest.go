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
		Forms      bool `yaml:"forms"`
		Storage    struct {
			Enabled    bool `yaml:"enabled"`
			ReadWrite  bool `yaml:"readwrite"`
			WholeStore bool `yaml:"wholestore"`
		} `yaml:"storage"`
		FileOpener map[string]string `yaml:"file_opener"`
	} `yaml:"app"`
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
	return out, nil
}

type Warning struct {
	error

	Fatal bool
}

func (m *Manifest) Warnings() <-chan Warning {
	ch := make(chan Warning)

	go func(ch chan<- Warning) {
		defer close(ch)

		{
			found := false
			for _, typ := range []string{"fileinfo", "storage", "app"} {
				if m.Type == typ {
					found = true
				}
			}
			if !found {
				ch <- Warning{error: fmt.Errorf("%s is not a valid type", m.Type), Fatal: true}
			}
		}

		if len(m.Permissions.App.FileOpener) > 0 {
			if !m.Permissions.App.Storage.Enabled {
				ch <- Warning{error: fmt.Errorf("Manifest has file openers specified, but storage library is disabled.")}
			} else if !m.Permissions.App.Storage.WholeStore {
				ch <- Warning{error: fmt.Errorf("Manifest has file openers specified, but will only have access to a subset. Which is currently not supported.")}
			}
		}
	}(ch)

	return ch
}

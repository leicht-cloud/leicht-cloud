package plugin

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManifestWarning(t *testing.T) {
	units := []struct {
		name             string
		input            string
		expectedWarnings []Warning
	}{
		{
			name: "invalid type",
			input: `
name: INVALID
type: INVALID
`,
			expectedWarnings: []Warning{
				{error: fmt.Errorf("INVALID is not a valid type"), Fatal: true},
			},
		},
		{
			name: "fileopener, with storage disabled",
			input: `
name: test
type: app
permissions:
  app:
    storage:
      enabled: false
    file_opener:
      text/*: /file?file=%file%
`,
			expectedWarnings: []Warning{
				{error: fmt.Errorf("Manifest has file openers specified, but storage library is disabled.")},
			},
		},
		{
			name: "fileopener, no wholestore",
			input: `
name: test
type: app
permissions:
  app:
    storage:
      enabled: true
      wholestore: false
    file_opener:
      text/*: /file?file=%file%
`,
			expectedWarnings: []Warning{
				{error: fmt.Errorf("Manifest has file openers specified, but will only have access to a subset. Which is currently not supported.")},
			},
		},
	}

	for _, unit := range units {
		t.Run(unit.name, func(t *testing.T) {
			manifest, err := parseManifest(strings.NewReader(unit.input))
			if err != nil {
				t.Fatal(err)
			}

			ch := manifest.Warnings()
			expected := unit.expectedWarnings

			for warning := range ch {
				if len(expected) == 0 {
					t.Errorf("Unexpected warning: %s", warning)
				} else {
					w := expected[0]
					expected = expected[1:]

					assert.Equal(t, w.Error(), warning.Error())
					assert.Equal(t, w.Fatal, warning.Fatal)
				}
			}

			assert.Empty(t, expected, "Didn't see all expected warnings")
		})
	}
}

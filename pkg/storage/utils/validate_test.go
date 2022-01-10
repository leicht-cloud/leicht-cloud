package utils

import (
	"testing"

	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage/builtin/memory"
	"github.com/stretchr/testify/assert"
)

func TestValidatePath(t *testing.T) {
	units := []struct {
		input  string
		output error
	}{
		{"input", nil},
		{"..", ErrDirectoryBack},
		{string("\x06"), ErrInvisibleCharacter},
		{"this should/be/a/valid path", nil},
	}

	for _, unit := range units {
		err := ValidatePath(unit.input)
		assert.Equal(t, unit.output, err)
	}
}

func TestWrappedValidate(t *testing.T) {
	provider := &ValidateWrapper{memory.NewStorageProvider()}

	storage.TestStorageProvider(provider, t)
}

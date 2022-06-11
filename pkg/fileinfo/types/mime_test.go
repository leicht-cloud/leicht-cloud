package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMime(t *testing.T) {
	units := []struct {
		in  string
		out MimeType
		err error
	}{
		{
			in: "text/plain",
			out: MimeType{
				Type:    "text",
				SubType: "plain",
			},
			err: nil,
		},
		{
			in:  "",
			err: ErrInvalidMime,
		},
	}

	for _, unit := range units {
		t.Run(unit.in, func(t *testing.T) {
			out, err := ParseMime(unit.in)
			assert.Equal(t, unit.err, err)
			if err == nil {
				assert.Equal(t, unit.out.Type, out.Type)
				assert.Equal(t, unit.out.SubType, out.SubType)
			}
		})
	}
}

func TestMimeMatch(t *testing.T) {
	units := []struct {
		pattern string
		mime    MimeType
		output  bool
	}{
		{
			pattern: "text/plain",
			mime: MimeType{
				Type:    "text",
				SubType: "plain",
			},
			output: true,
		},
		{
			pattern: "text/*",
			mime: MimeType{
				Type:    "text",
				SubType: "plain",
			},
			output: true,
		},
		{
			pattern: "image/*",
			mime: MimeType{
				Type:    "text",
				SubType: "plain",
			},
			output: false,
		},
	}

	for _, unit := range units {
		t.Run(unit.pattern, func(t *testing.T) {
			out := unit.mime.Match(unit.pattern)
			assert.Equal(t, unit.output, out)
		})
	}
}

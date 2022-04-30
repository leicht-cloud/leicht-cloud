package types

import (
	"fmt"
	"strings"
)

type MimeType struct {
	Type    string `json:"type"`
	SubType string `json:"subtype"`
}

func ParseMime(in string) (MimeType, error) {
	split := strings.SplitN(in, "/", 2)

	return MimeType{
		Type:    split[0],
		SubType: split[1],
	}, nil
}

func (m MimeType) String() string {
	return fmt.Sprintf("%s/%s", m.Type, m.SubType)
}

func (m MimeType) Match(pattern string) bool {
	parsed, err := ParseMime(pattern)
	if err != nil {
		return false
	}
	if parsed.Type != m.Type {
		return false
	}
	if parsed.SubType == "*" {
		return true
	} else if parsed.SubType == m.SubType {
		return true
	}

	return false
}

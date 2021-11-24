package main

import (
	"testing"

	"github.com/schoentoon/go-cloud/pkg/fileinfo"
)

func TestFileInfoImageSize(t *testing.T) {
	provider := &ImageSize{}
	fileinfo.RunFileInfoProviderFromFile(provider, "testdata/sample.jpg", t, []byte(`{"width":600,"height":400}`))
}

func BenchmarkFileInfoImageSize(b *testing.B) {
	provider := &ImageSize{}
	fileinfo.RunFileInfoProviderFromFile(provider, "testdata/sample.jpg", b, []byte(`{"width":600,"height":400}`))
}

package main

import (
	"os"

	"github.com/schoentoon/go-cloud/pkg/storage/local"
	"github.com/schoentoon/go-cloud/pkg/storage/plugin"
)

func main() {
	// TODO: We need some way to pass config options to external plugins like this one
	tmp, err := os.MkdirTemp("/tmp", "locallol.")
	if err != nil {
		panic(err)
	}
	provider := &local.StorageProvider{RootPath: tmp}

	err = plugin.Start(provider)
	if err != nil {
		panic(err)
	}
}

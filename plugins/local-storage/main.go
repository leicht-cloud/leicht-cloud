package main

import (
	"github.com/schoentoon/go-cloud/pkg/storage/local"
	"github.com/schoentoon/go-cloud/pkg/storage/plugin"
)

func main() {
	provider := &local.StorageProvider{RootPath: "./tmp"}

	err := plugin.Start(provider)
	if err != nil {
		panic(err)
	}
}

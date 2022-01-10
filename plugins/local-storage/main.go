package main

import (
	"github.com/leicht-cloud/leicht-cloud/pkg/storage/builtin/local"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage/plugin"
)

func main() {
	provider := &local.StorageProvider{}

	err := plugin.Start(provider)
	if err != nil {
		panic(err)
	}
}

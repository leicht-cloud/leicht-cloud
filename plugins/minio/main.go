package main

import (
	"github.com/schoentoon/go-cloud/pkg/storage/plugin"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	provider := &StorageProvider{}

	err := plugin.Start(provider)
	if err != nil {
		panic(err)
	}
}

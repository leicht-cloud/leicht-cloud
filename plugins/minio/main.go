package main

import (
	"github.com/leicht-cloud/leicht-cloud/pkg/storage/plugin"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})

	provider := &StorageProvider{}

	err := plugin.Start(provider)
	if err != nil {
		panic(err)
	}
}

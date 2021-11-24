package main

import (
	"github.com/schoentoon/go-cloud/pkg/fileinfo/plugin"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})

	provider := &ImageSize{}

	err := plugin.Start(provider)
	if err != nil {
		panic(err)
	}
}

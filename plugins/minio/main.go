package main

import (
	"github.com/schoentoon/go-cloud/pkg/storage/plugin"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	// TODO: Handle this through config
	provider := &StorageProvider{
		Hostname:        "127.0.0.1:9000",
		Https:           false,
		AccessKey:       "go-cloud",
		SecretAccessKey: "ThisIsASecret",
		Prefix:          "cloud-",
	}

	err := plugin.Start(provider)
	if err != nil {
		panic(err)
	}
}

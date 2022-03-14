package plugin

import (
	"github.com/leicht-cloud/leicht-cloud/pkg/plugin/common"
)

// This is meant to be called in the main() of your plugin
func Start() (err error) {
	return common.Init(nil)
}

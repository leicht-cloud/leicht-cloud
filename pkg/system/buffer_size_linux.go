package system

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// This is mostly based on the information found in this stackexchange post
// https://unix.stackexchange.com/questions/424380/what-values-may-linux-use-for-the-default-unix-socket-buffer-size
// On both amd64, aarch64 and x86 the value in rmem_default is larger than 1024 * 1024
// so having this as the default is probably a good bet. We however do try to read
// the value from rmem_default anyway. It isn't 100% clear to me whether this is the buffer size
// for unix sockets as well. It is however clear that performance is instantly better with this
// value. If needed it could always be tweaked later on of course.

var buffer_size int64 = 1024 * 1024

const error_message = "Error while trying to determine buffer size: %s"

func init() {
	// we could also consider reading the wmem_default?
	f, err := os.Open("/proc/sys/net/core/rmem_default")
	if err != nil {
		logrus.Errorf(error_message, err)
		return
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		logrus.Errorf(error_message, err)
		return
	}
	raw := strings.Trim(string(data), "\n")

	buffer_size, err = strconv.ParseInt(raw, 10, 64)
	if err != nil {
		logrus.Errorf(error_message, err)
		return
	}
}

func GetBufferSize() int64 {
	return buffer_size
}

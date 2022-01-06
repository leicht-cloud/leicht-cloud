package limiter

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReader(t *testing.T) {
	// our buffer has 15 bytes, our reader is rate limited to 5 bytes, with a burst of 5.
	// meaning it should take 2 seconds, so we really just test whether
	// it will take more than 1 second or not
	buf := bytes.NewBufferString("123456789012345")
	bufLen := int64(buf.Len())
	reader := NewReader(buf, 5, 5)
	defer reader.Close()

	start := time.Now()

	n, err := io.Copy(io.Discard, reader)
	assert.NoError(t, err)
	assert.Equal(t, bufLen, n)

	end := time.Now()
	duration := end.Sub(start)

	assert.Greater(t, duration, time.Second)
}

func TestReaderDevZero(t *testing.T) {
	f, err := os.Open("/dev/zero")
	if err != nil && os.IsNotExist(err) {
		t.Skip(err)
	}
	defer f.Close()

	// as /dev/zero is infinite, we first limit the reader to 10 megabytes
	// after this we set up a rate limited reader with a rate limit of 2 megabytes
	// and a burst of 5 megabyte, this should take about 3 seconds
	reader := io.LimitReader(f, 1024*1024*10)
	ratelimited := NewReader(reader, 1024*1024*2, 1024*1024*4)

	start := time.Now()

	n, err := io.Copy(io.Discard, ratelimited)
	assert.NoError(t, err)
	assert.Greater(t, n, int64(0))

	end := time.Now()
	duration := end.Sub(start)

	assert.Greater(t, duration, time.Second)
}

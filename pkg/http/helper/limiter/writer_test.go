package limiter

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResponseWriter(t *testing.T) {
	// our buffer has 15 bytes, our reader is rate limited to 5 bytes, with a burst of 5.
	// meaning it should take 2 seconds, so we really just test whether
	// it will take more than 1 second or not
	buf := bytes.NewBufferString("123456789012345")
	buflen := int64(buf.Len())
	writer := NewResponseWriter(io.Discard, 5, 5)

	start := time.Now()

	n, err := io.Copy(writer, buf)
	assert.NoError(t, err)
	assert.Equal(t, buflen, n)

	end := time.Now()
	duration := end.Sub(start)

	assert.Greater(t, duration, time.Second)
}

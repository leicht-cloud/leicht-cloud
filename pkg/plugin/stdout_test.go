package plugin

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testText = []byte("This is just a simple test\n")

func TestStdoutChannel(t *testing.T) {
	stdout := newStdout()

	// first test just requesting a single channel and removing it again
	assert.Len(t, stdout.channels, 0)
	ch := stdout.Channel()
	assert.Len(t, stdout.channels, 1)
	assert.Len(t, ch.ch, 0)
	ch.Close()
	assert.Len(t, stdout.channels, 0)

	// then we request a new one and actually write to it
	ch = stdout.Channel()
	assert.Len(t, stdout.channels, 1)
	assert.Len(t, ch.ch, 0)
	n, err := stdout.Write(testText)
	assert.Equal(t, len(testText), n)
	assert.NoError(t, err)
	assert.Len(t, ch.ch, 1)
	assert.Equal(t, testText, <-ch.Channel())
	ch.Close()
	assert.Len(t, stdout.channels, 0)

	// finally, we request another one. but as stdout now has data we expect initial data
	ch = stdout.Channel()
	assert.Len(t, stdout.channels, 1)
	assert.Len(t, ch.ch, 1)
	assert.Equal(t, testText, <-ch.Channel())
	ch.Close()
	assert.Len(t, stdout.channels, 0)
}

func TestRemoveChannel(t *testing.T) {
	stdout := newStdout()

	assert.Len(t, stdout.channels, 0)
	for i := 0; i < 100; i++ {
		stdout.Channel()
	}
	assert.Len(t, stdout.channels, 100)

	wg := sync.WaitGroup{}
	wg.Add(1)

	// we do the deletion in another goroutine, while we write to it
	// as this could trigger a datarace, thus this test is working as a test for this
	ch := stdout.channels[50]
	go func() {
		ch.Close()
		wg.Done()
	}()
	stdout.Write(testText)
	wg.Wait()

	assert.Len(t, stdout.channels, 99)
	assert.NotContains(t, stdout.channels, ch)
}

func TestReader(t *testing.T) {
	stdout := newStdout()

	stdout.Write(testText)
	reader := stdout.Reader()

	buf := make([]byte, 1024*4)
	n, err := reader.Read(buf)
	assert.NoError(t, err)
	assert.Len(t, testText, n)

	stdout.Write(testText)
	assert.NoError(t, err)
	assert.Len(t, testText, n)
}

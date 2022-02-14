package plugin

import (
	"bytes"
	"sync"
)

// TODO: stdout of plugins is currently kept forever, we'll probably want to put a limit on this and purge early lines

type Stdout struct {
	buffer bytes.Buffer

	mutex    sync.RWMutex
	channels []*StdoutChannel
}

func newStdout() *Stdout {
	return &Stdout{
		channels: make([]*StdoutChannel, 0),
	}
}

func (s *Stdout) Bytes() []byte {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.buffer.Bytes()
}

type StdoutChannel struct {
	ch chan []byte

	stdout *Stdout
}

func (s *Stdout) Write(d []byte) (int, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, c := range s.channels {
		select {
		case c.ch <- d:
		default:
		}
	}

	return s.buffer.Write(d)
}

func (s *Stdout) Channel() *StdoutChannel {
	out := &StdoutChannel{
		ch:     make(chan []byte, 8),
		stdout: s,
	}
	if s.buffer.Len() > 0 {
		out.ch <- s.buffer.Bytes()
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.channels = append(s.channels, out)
	return out
}

func (s *Stdout) removeChannel(ch *StdoutChannel) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for index, c := range s.channels {
		if c == ch {
			out := make([]*StdoutChannel, 0, len(s.channels)-1)
			out = append(out, s.channels[:index]...)
			out = append(out, s.channels[index+1:]...)
			s.channels = out
			return
		}
	}
}

func (c *StdoutChannel) Channel() <-chan []byte {
	return c.ch
}

func (c *StdoutChannel) Close() error {
	c.stdout.removeChannel(c)
	close(c.ch)
	return nil
}

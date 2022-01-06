package limiter

import (
	"io"

	"github.com/juju/ratelimit"
)

type reader struct {
	wrapped io.Reader
	bucket  *ratelimit.Bucket
}

func NewReader(wrapped io.Reader, rate float64, burst int64) io.ReadCloser {
	return &reader{
		wrapped: wrapped,
		bucket:  ratelimit.NewBucketWithRate(rate, burst),
	}
}

func (r *reader) Read(p []byte) (int, error) {
	n, err := r.wrapped.Read(p)
	if n <= 0 {
		return n, err
	}
	r.bucket.Wait(int64(n))
	return n, err
}

func (r *reader) Close() error {
	closer, ok := r.wrapped.(io.Closer)
	if ok {
		return closer.Close()
	}
	return nil
}

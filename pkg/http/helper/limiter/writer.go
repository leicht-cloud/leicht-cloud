package limiter

import (
	"io"
	"net/http"

	"github.com/juju/ratelimit"
)

type responseWriter struct {
	wrapped io.Writer
	bucket  *ratelimit.Bucket
}

func NewResponseWriter(wrapped io.Writer, rate float64, burst int64) http.ResponseWriter {
	return &responseWriter{
		wrapped: wrapped,
		bucket:  ratelimit.NewBucketWithRate(rate, burst),
	}
}

func (w *responseWriter) Write(p []byte) (int, error) {
	n, err := w.wrapped.Write(p)
	if n <= 0 {
		return n, err
	}
	w.bucket.Wait(int64(n))
	return n, err
}

func (w *responseWriter) Header() http.Header {
	wrapped, ok := w.wrapped.(http.ResponseWriter)
	if ok {
		return wrapped.Header()
	}

	panic("wrapped writer isn't a http.ResponseWriter")
}

func (w *responseWriter) WriteHeader(statusCode int) {
	wrapped, ok := w.wrapped.(http.ResponseWriter)
	if ok {
		wrapped.WriteHeader(statusCode)
		return
	}

	panic("wrapped writer isn't a http.ResponseWriter")
}

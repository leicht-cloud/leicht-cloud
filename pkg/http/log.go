package http

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/schoentoon/go-cloud/pkg/auth"
	"github.com/sirupsen/logrus"
)

type (
	// struct for holding response details
	responseData struct {
		status int
		size   int
	}

	// our http.ResponseWriter implementation
	loggingResponseWriter struct {
		http.ResponseWriter // compose original http.ResponseWriter
		responseData        *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b) // write response using original http.ResponseWriter
	r.responseData.size += size            // capture size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode) // write status code using original http.ResponseWriter
	r.responseData.status = statusCode       // capture status code
}

func (r *loggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijack not supported")
	}
	return h.Hijack()
}

func WithLogging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		start := time.Now()

		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lrw := loggingResponseWriter{
			ResponseWriter: rw, // compose original http.ResponseWriter
			responseData:   responseData,
		}
		h.ServeHTTP(&lrw, req) // inject our implementation of http.ResponseWriter

		duration := time.Since(start)

		user := auth.GetUserFromRequest(req)

		entry := logrus.WithFields(logrus.Fields{
			"src":      req.RemoteAddr,
			"uri":      req.RequestURI,
			"method":   req.Method,
			"status":   responseData.status,
			"duration": duration,
			"size":     responseData.size,
		})
		if user != nil {
			entry = entry.WithField("user", user.Email)
		}
		entry.Info("request completed")
	})
}

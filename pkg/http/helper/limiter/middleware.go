package limiter

import "net/http"

func Middleware(rate float64, burst int64, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = NewReader(r.Body, rate, burst)

		handler.ServeHTTP(w, r)
	})
}

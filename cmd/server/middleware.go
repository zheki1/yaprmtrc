package main

import (
	"net/http"
	"time"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.status = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	if lrw.status == 0 {
		lrw.status = http.StatusOK
	}
	n, err := lrw.ResponseWriter.Write(b)
	lrw.size += n
	return n, err
}

func LoggingMiddleware(logger Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			lrw := &loggingResponseWriter{ResponseWriter: w}
			next.ServeHTTP(lrw, r)

			logger.Infow(
				"http request",
				"method", r.Method,
				"uri", r.RequestURI,
				"duration", time.Since(start),
				"status", lrw.status,
				"size", lrw.size,
			)
		})
	}
}

package main

import (
	"bytes"
	"net/http"
	"strings"
	"time"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
	body   bytes.Buffer
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.status = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	if lrw.status == 0 {
		lrw.status = http.StatusOK
	}
	lrw.body.Write(b) // ⬅ сохраняем тело ответа
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

			fields := []any{
				"method", r.Method,
				"uri", r.RequestURI,
				"duration", time.Since(start),
				"status", lrw.status,
				"size", lrw.size,
			}

			if lrw.status >= http.StatusBadRequest {
				fields = append(fields, "error", strings.TrimSpace(lrw.body.String()))
				logger.Infow("http error", fields...)
				return
			}

			logger.Infow("http request", fields...)
		})
	}
}

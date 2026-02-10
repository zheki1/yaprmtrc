package main

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/zheki1/yaprmtrc/internal/security"
)

func HashMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if key == "" {
				next.ServeHTTP(writer, request)
				return
			}

			got := request.Header.Get("HashSHA256")
			if got != "" {
				body, err := io.ReadAll(request.Body)
				if err != nil {
					http.Error(writer, "bad request", http.StatusBadRequest)
					return
				}
				_ = request.Body.Close()
				computed := security.CalcHash(body, key)
				if !strings.EqualFold(got, computed) {
					//http.Error(writer, "bad hash", http.StatusBadRequest)
					//return
				}
				request.Body = io.NopCloser(bytes.NewReader(body))
			}

			rec := NewRecorder(writer)
			next.ServeHTTP(rec, request)

			if key != "" {
				sum := security.CalcHash(rec.Body(), key)
				rec.Header().Set("HashSHA256", sum)
			}
			rec.FlushTo(writer)

		})
	}
}

type ResponseRecorder struct {
	header      http.Header
	status      int
	body        bytes.Buffer
	wroteHeader bool
}

func NewRecorder(_ http.ResponseWriter) *ResponseRecorder {
	return &ResponseRecorder{
		header: make(http.Header),
	}
}

func (r *ResponseRecorder) Header() http.Header {
	return r.header
}

func (r *ResponseRecorder) WriteHeader(statusCode int) {
	if r.wroteHeader {
		return
	}
	r.status = statusCode
	r.wroteHeader = true
}

func (r *ResponseRecorder) Write(p []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.body.Write(p)
}

func (r *ResponseRecorder) Body() []byte {
	return r.body.Bytes()
}

func (r *ResponseRecorder) FlushTo(w http.ResponseWriter) {
	for k, vv := range r.header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	if r.status == 0 {
		r.status = http.StatusOK
	}
	w.WriteHeader(r.status)

	if r.body.Len() > 0 {
		_, _ = w.Write(r.body.Bytes())
	}
}

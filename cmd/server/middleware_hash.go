package main

import (
	"bytes"
	"io"
	"net/http"

	"github.com/zheki1/yaprmtrc.git/internal/utils"
)

func (s *Server) hashRequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if s.key == "" {
			next.ServeHTTP(w, r)
			return
		}

		headerHash := r.Header.Get("HashSHA256")
		if headerHash == "" {
			http.Error(w, "missing hash", http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		r.Body.Close()

		expected := utils.CalculateHMAC(body, s.key)

		if headerHash != expected {
			http.Error(w, "invalid hash", http.StatusBadRequest)
			return
		}

		r.Body = io.NopCloser(bytes.NewReader(body))

		next.ServeHTTP(w, r)
	})
}

type hashResponseWriter struct {
	http.ResponseWriter
	body bytes.Buffer
}

func (w *hashResponseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (s *Server) hashResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.key == "" {
			next.ServeHTTP(w, r)
			return
		}

		hw := &hashResponseWriter{ResponseWriter: w}
		next.ServeHTTP(hw, r)

		hash := utils.CalculateHMAC(hw.body.Bytes(), s.key)
		w.Header().Set("HashSHA256", hash)
	})
}

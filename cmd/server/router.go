package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func router(s *Server) http.Handler {
	r := chi.NewRouter()

	r.Use(LoggingMiddleware(s.logger))
	r.Use(GzipMiddleware)
	r.Use(middleware.StripSlashes)

	r.Post("/update/{type}/{name}/{value}", s.updateHandler)
	r.Post("/update", s.updateHandlerJSON)
	r.Post("/value", s.valueHandlerJSON)
	r.Get("/value/{type}/{name}", s.valueHandler)
	r.Get("/", s.pageHandler)
	r.Get("/ping", s.pingHandler)
	r.Post("/updates", s.batchUpdateHandler)

	return r
}

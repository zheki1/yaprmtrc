package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func router(server *Server, logger Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(LoggingMiddleware(logger))
	r.Use(GzipMiddleware)
	r.Use(middleware.StripSlashes)

	r.Post("/update/{type}/{name}/{value}", server.updateHandler)
	r.Post("/update", server.updateHandlerJSON)
	r.Post("/value", server.valueHandlerJSON)
	r.Get("/value/{type}/{name}", server.valueHandler)
	r.Get("/", server.pageHandler)

	return r
}

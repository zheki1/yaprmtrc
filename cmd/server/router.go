package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func router(storage Storage, logger Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(LoggingMiddleware(logger))

	s := &Server{storage: storage}

	r.Post("/update/{type}/{name}/{value}", s.updateHandler())
	r.Get("/value/{type}/{name}", s.valueHandler())
	r.Get("/", s.pageHandler())

	return r
}

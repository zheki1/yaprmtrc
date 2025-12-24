package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	//"github.com/go-chi/chi/v5/middleware"
)

func router(storage Storage, logger Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(LoggingMiddleware(logger))
	//r.Use(middleware.StripSlashes)

	s := &Server{storage: storage}

	r.Post("/update/{type}/{name}/{value}", s.updateHandler)
	r.Post("/update", s.updateHandlerJSON)
	r.Post("/update/", s.updateHandlerJSON)
	r.Post("/value", s.valueHandlerJSON)
	r.Post("/value/", s.valueHandlerJSON)
	r.Get("/value/{type}/{name}", s.valueHandler)
	r.Get("/", s.pageHandler)

	return r
}

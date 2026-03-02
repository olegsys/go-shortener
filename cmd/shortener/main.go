package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/olegsys/go-shortener/internal/handler"
	"github.com/olegsys/go-shortener/internal/model"
)

func main() {
	store := model.NewStore()
	h := &handler.Handler{Store: store}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Post("/", h.Shorten)
	r.Get("/{id}", h.Redirect)
	err := http.ListenAndServe(`:8080`, r)
	if err != nil {
		panic(err)
	}
}

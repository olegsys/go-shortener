package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/olegsys/go-shortener/internal/config"
	"github.com/olegsys/go-shortener/internal/handler"
	"github.com/olegsys/go-shortener/internal/model"
)

func main() {
	store := model.NewStore()
	cfg := config.LoadConfig()
	h := handler.NewHandler(store, cfg)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Post("/", h.Shorten)
	r.Get("/{id}", h.Redirect)

	fmt.Println("Listen on:", cfg.ListenAddress)
	err := http.ListenAndServe(cfg.ListenAddress, r)
	if err != nil {
		panic(err)
	}
}

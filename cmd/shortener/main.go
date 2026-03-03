package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/olegsys/go-shortener/internal/config"
	"github.com/olegsys/go-shortener/internal/handler"
	"github.com/olegsys/go-shortener/internal/repository"
	"github.com/olegsys/go-shortener/internal/service"
)

func main() {
	cfg := config.LoadConfig()
	storage := repository.NewMapStorage()
	shortenerService := service.NewShortenerService(storage, cfg.BaseUrl)
	urlHandler := handler.NewHandler(shortenerService)

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Post("/", urlHandler.Shorten)
	router.Get("/{id}", urlHandler.Redirect)

	fmt.Println("Listen on:", cfg.ListenAddress)
	err := http.ListenAndServe(cfg.ListenAddress, router)
	if err != nil {
		panic(err)
	}
}

package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/olegsys/go-shortener/internal/config"
	"github.com/olegsys/go-shortener/internal/handler"
	"github.com/olegsys/go-shortener/internal/middleware"
	"github.com/olegsys/go-shortener/internal/repository"
	"github.com/olegsys/go-shortener/internal/service"
	"go.uber.org/zap"
)

func main() {
	cfg := config.LoadConfig()
	logger, err := zap.NewProduction()
	if err != nil {
		panic("cannot initialize zap")
	}
	defer logger.Sync()

	storage := repository.NewMapStorage()
	shortenerService := service.NewShortenerService(storage, cfg.BaseURL)
	urlHandler := handler.NewHandler(shortenerService)

	router := chi.NewRouter()
	router.Use(middleware.Logging(logger))
	router.Post("/", urlHandler.Shorten)
	router.Get("/{id}", urlHandler.Redirect)

	fmt.Println("Listen on:", cfg.ListenAddress)
	fmt.Println("Base URL:", cfg.BaseURL)
	err = http.ListenAndServe(cfg.ListenAddress, router)
	if err != nil {
		panic(err)
	}
}

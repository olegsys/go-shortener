package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
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

	var dbStorage *repository.PostgresStorage
	if cfg.DatabaseDSN != "" {
		dbStorage, err = repository.NewPostgresStorage(cfg.DatabaseDSN)
		if err != nil {
			logger.Fatal("Failed to connect to database",
				zap.Error(err),
			)
		}
		defer dbStorage.Close(context.Background())
		logger.Info("Connected to PostgreSQL database")
	}

	storage := repository.NewMapStorage()
	shortenerService := service.NewShortenerService(storage, cfg.BaseURL)
	urlHandler := handler.NewHandler(shortenerService)
	pingHandler := handler.NewPingHandler(dbStorage)
	router := chi.NewRouter()

	err = storage.LoadFromFile(cfg.StorageFile)
	if err != nil {
		logger.Error("Error loading data from file",
			zap.Error(err),
		)
	}

	router.Use(middleware.Logging(logger))
	router.Use(chimw.Compress(5, "application/json", "text/html"))
	router.Use(middleware.DecompressMiddleware)
	router.Post("/", urlHandler.Shorten)
	router.Post("/api/shorten", urlHandler.ShortenJson)
	router.Get("/{id}", urlHandler.Redirect)
	router.Get("/ping", pingHandler.Ping)

	srv := &http.Server{
		Addr:    cfg.ListenAddress,
		Handler: router,
	}
	go func() {
		logger.Info("Server startup params",
			zap.String("Listen on:", cfg.ListenAddress),
			zap.String("Base URL:", cfg.BaseURL),
			zap.String("File storage path:", cfg.StorageFile),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server listen failed",
				zap.Error(err),
			)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	logger.Info("Shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := storage.SaveToFile(cfg.StorageFile); err != nil {
		logger.Error("File with data not saved",
			zap.String("Filepath", cfg.StorageFile),
			zap.Error(err),
		)
	}
	if dbStorage != nil {
		if err := dbStorage.Close(ctx); err != nil {
			logger.Error("Database close failed",
				zap.Error(err),
			)
		}
	}
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Graceful shutdown failed",
			zap.Error(err),
		)
		srv.Close()
	}
	logger.Info("Server stopped gracefully")
}

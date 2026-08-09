package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	shortenerpb "github.com/olegsys/go-shortener/api/shortener/v1"
	"github.com/olegsys/go-shortener/internal/audit"
	"github.com/olegsys/go-shortener/internal/config"
	"github.com/olegsys/go-shortener/internal/grpcserver"
	"github.com/olegsys/go-shortener/internal/handler"
	"github.com/olegsys/go-shortener/internal/middleware"
	"github.com/olegsys/go-shortener/internal/repository"
	"github.com/olegsys/go-shortener/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

// Глобальные переменные для информации о сборке
var buildVersion string
var buildDate string
var buildCommit string

func main() {
	if buildVersion == "" {
		buildVersion = "N/A"
	}
	if buildDate == "" {
		buildDate = "N/A"
	}
	if buildCommit == "" {
		buildCommit = "N/A"
	}

	// Выводим информацию о сборке в stdout
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)

	logger, err := zap.NewProduction()
	if err != nil {
		panic("cannot initialize zap")
	}
	defer func() { _ = logger.Sync() }()

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	var dbStorage *repository.PostgresStorage
	var mapStorage *repository.MapStorage
	var storage service.URLStore

	if cfg.DatabaseDSN != "" {
		dbStorage, err = repository.NewPostgresStorage(cfg.DatabaseDSN)
		if err != nil {
			logger.Panic("Failed to connect to database",
				zap.Error(err),
			)
		}
		logger.Info("Connected to PostgreSQL database")
		storage = dbStorage
	} else if cfg.StorageFile != "" {
		mapStorage = repository.NewMapStorage()
		if err := mapStorage.LoadFromFile(cfg.StorageFile); err != nil {
			logger.Error("Error loading data from file",
				zap.Error(err),
			)
		}
		storage = mapStorage
		logger.Info("Using file storage")
	} else {
		storage = repository.NewMapStorage()
		logger.Info("Using in-memory storage")
	}

	shortenerService := service.NewShortenerService(storage, cfg.BaseURL)
	deletionSvc := service.NewDeletionService(shortenerService)

	var auditor audit.Auditor
	var auditBus *audit.EventBus

	if cfg.AuditFile != "" || cfg.AuditURL != "" {
		auditBus = audit.NewEventBus()

		if cfg.AuditFile != "" {
			fileObs, err := audit.NewFileObserver(cfg.AuditFile)
			if err != nil {
				logger.Error("Failed to init file audit observer", zap.Error(err))
			} else {
				auditBus.Register(fileObs)
				logger.Info("File audit observer enabled", zap.String("path", cfg.AuditFile))
			}
		}
		if cfg.AuditURL != "" {
			httpObs := audit.NewHTTPObserver(cfg.AuditURL)
			auditBus.Register(httpObs)
			logger.Info("HTTP audit observer enabled", zap.String("url", cfg.AuditURL))
		}

		auditor = auditBus
	} else {
		auditor = &audit.NoopAuditor{}
		logger.Info("Audit is disabled, using NoopAuditor")
	}

	var grpcSrv *grpc.Server
	var grpcLis net.Listener

	if cfg.GRPCListenAddress != "" {
		grpcOpts := []grpc.ServerOption{
			grpc.ChainUnaryInterceptor(
				middleware.GRPCLogging(logger),
				middleware.GRPCAuthInterceptor(cfg.SecretKey),
			),
		}

		if cfg.EnableHTTPS {
			cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
			if err != nil {
				logger.Panic("failed to load TLS certificate for gRPC",
					zap.Error(err),
				)
			}

			tlsConfig := &tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS12,
			}

			grpcOpts = append(grpcOpts, grpc.Creds(credentials.NewTLS(tlsConfig)))
		}

		grpcSrv = grpc.NewServer(grpcOpts...)

		shortenerpb.RegisterShortenerServiceServer(
			grpcSrv,
			grpcserver.New(shortenerService, auditor),
		)

		// Для отладки через grpcurl
		reflection.Register(grpcSrv)

		grpcLis, err = net.Listen("tcp", cfg.GRPCListenAddress)
		if err != nil {
			logger.Panic("gRPC listen failed",
				zap.Error(err),
			)
		}
	}

	urlHandler := handler.NewHandler(shortenerService, deletionSvc, auditor)
	pingHandler := handler.NewPingHandler(dbStorage)
	router := chi.NewRouter()

	router.Use(middleware.Logging(logger))
	router.Use(middleware.AuthMiddleware(cfg.SecretKey, cfg.EnableHTTPS))
	router.Use(chimw.Compress(5, "application/json", "text/html"))
	router.Use(middleware.DecompressMiddleware)
	router.Post("/", urlHandler.Shorten)
	router.Post("/api/shorten", urlHandler.ShortenJSON)
	router.Post("/api/shorten/batch", urlHandler.ShortenBatchJSON)
	router.Get("/{id}", urlHandler.Redirect)
	router.Get("/api/user/urls", urlHandler.GetUserURLs)
	router.Get("/ping", pingHandler.Ping)
	router.Delete("/api/user/urls", urlHandler.DeleteURLs)
	router.With(middleware.TrustedSubnetMiddleware(cfg.TrustedSubnet)).Get("/api/internal/stats", urlHandler.Stats)
	// Endpoints для pprof
	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	router.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	router.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
	router.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))

	srv := &http.Server{
		Addr:    cfg.ListenAddress,
		Handler: router,
	}

	httpLis, err := net.Listen("tcp", cfg.ListenAddress)
	if err != nil {
		if grpcLis != nil {
			_ = grpcLis.Close()
		}

		logger.Panic("HTTP listen failed",
			zap.Error(err),
		)
	}

	if grpcSrv != nil {
		go func() {
			logger.Info("gRPC server startup params",
				zap.String("listen_on", cfg.GRPCListenAddress),
				zap.Bool("tls_enabled", cfg.EnableHTTPS),
			)

			if err := grpcSrv.Serve(grpcLis); err != nil {
				logger.Panic("gRPC serve failed",
					zap.Error(err),
				)
			}
		}()
	}

	go func() {
		logger.Info("Server startup params",
			zap.String("Listen on:", cfg.ListenAddress),
			zap.String("Base URL:", cfg.BaseURL),
			zap.Bool("HTTPS enabled:", cfg.EnableHTTPS),
		)
		var err error
		if cfg.EnableHTTPS {
			if cfg.CertFile == "" || cfg.KeyFile == "" {
				logger.Panic("TLS enabled but certificate/key paths are not configured")
			}

			if _, statErr := os.Stat(cfg.CertFile); statErr != nil {
				logger.Panic("TLS certificate file is not available",
					zap.String("cert_file", cfg.CertFile),
					zap.Error(statErr),
				)
			}

			if _, statErr := os.Stat(cfg.KeyFile); statErr != nil {
				logger.Panic("TLS private key file is not available",
					zap.String("key_file", cfg.KeyFile),
					zap.Error(statErr),
				)
			}

			err = srv.ServeTLS(httpLis, cfg.CertFile, cfg.KeyFile)
		} else {
			err = srv.Serve(httpLis)
		}

		if err != nil && err != http.ErrServerClosed {
			logger.Panic("Server listen failed",
				zap.Error(err),
			)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	<-stop
	logger.Info("Shutdown signal received")

	// останавливаем HTTP-сервер
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Graceful shutdown failed", zap.Error(err))
		_ = srv.Close()
	}

	// останавливаем gRPC сервер
	if grpcSrv != nil {
		grpcSrv.GracefulStop()
		logger.Info("gRPC server stopped")
	}

	// останавливаем фоновый сервис удалений
	deletionSvc.Shutdown()

	// закрываем аудит
	if auditBus != nil {
		auditBus.Close()
	}

	// если используется файловое хранилище, то сохраняем его
	if mapStorage != nil {
		if err := mapStorage.SaveToFile(cfg.StorageFile); err != nil {
			logger.Error("File with data not saved",
				zap.String("Filepath", cfg.StorageFile),
				zap.Error(err),
			)
		}
	}
	// если используется PostgreSQL, то закрываем его
	if dbStorage != nil {
		if err := dbStorage.Close(ctx); err != nil {
			logger.Error("Database close failed",
				zap.Error(err),
			)
		}
	}

	logger.Info("Server stopped gracefully")
}

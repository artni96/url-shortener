package main

import (
	"database/sql"
	"net/http"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/handler/healthcheck"
	"github.com/artni96/url-shortener/internal/handler/urls"
	"github.com/artni96/url-shortener/internal/logger"
	"github.com/artni96/url-shortener/internal/repository"
	"github.com/artni96/url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func run(cfg *config.Config) error {

	appLogger, err := logger.InitLogger(cfg.DebugLevel)

	db, err := sql.Open("pgx", cfg.DatabaseDsn)
	defer db.Close()
	if err != nil {
		appLogger.Error("failed to connect to database",
			zap.String("database destination", cfg.DatabaseDsn),
			zap.String("error message", err.Error()),
		)
	}
	if err != nil {
		return err
	}
	urlRepository, err := repository.NewURLRepository(cfg, appLogger)
	if err != nil {
		return err
	}
	urlService := service.NewURLService(urlRepository, cfg)

	mainRouter := chi.NewRouter()
	urlRouter := urls.URLRouter(cfg, urlService, appLogger)
	healthRouter := healthcheck.HealthCheckRouter(db, appLogger)
	mainRouter.Mount("/", urlRouter)
	mainRouter.Mount("/ping", healthRouter)

	appLogger.Info("Starting server",
		zap.String("server address", cfg.ServerAddress),
	)
	return http.ListenAndServe(cfg.ServerAddress, mainRouter)
}

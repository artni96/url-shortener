package main

import (
	"context"
	"database/sql"
	"net/http"
	"os/exec"

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
	ctx := context.Background()
	appLogger, err := logger.InitLogger(cfg.DebugLevel)

	db, err := sql.Open("pgx", cfg.DatabaseDsn)
	defer db.Close()
	if err != nil {
		appLogger.Error("failed to connect to database",
			zap.String("database destination", cfg.DatabaseDsn),
			zap.String("error message", err.Error()),
		)
		return err
	}

	if err := runMigrations(db, ctx); err != nil && cfg.DatabaseDsn != "" {
		appLogger.Error("failed to run migrations",
			zap.String("error message", err.Error()),
		)
		return err
	} else {
		appLogger.Info("Migrations completed successfully")
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

func runMigrations(db *sql.DB, ctx context.Context) error {
	err := db.PingContext(ctx)
	if err != nil {
		return err
	}
	cmd := exec.Command(
		"migrate",
		"-database", "postgres://postgres:postgres@localhost:5432/url_shortener?sslmode=disable",
		"-path", "./migrations", "up",
	)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}

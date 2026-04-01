package main

import (
	"context"
	"net/http"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/config/db"
	"github.com/artni96/url-shortener/internal/handler/healthcheck"
	"github.com/artni96/url-shortener/internal/handler/urls"
	"github.com/artni96/url-shortener/internal/logger"
	"github.com/artni96/url-shortener/internal/repository"
	"github.com/artni96/url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func run(cfg *config.Config) error {
	NewApp := &config.App{
		DB:  nil,
		Cfg: cfg,
	}

	ctx := context.Background()
	appLogger, err := logger.InitLogger(NewApp.Cfg.DebugLevel)

	DBCon, err := db.InitDBConnection(ctx, NewApp.Cfg, appLogger)
	defer DBCon.Close()

	if DBCon != nil {
		NewApp.DB = DBCon
	}

	urlRepository, err := repository.NewURLRepository(&ctx, NewApp, appLogger)
	if err != nil {
		return err
	}
	urlService := service.NewURLService(urlRepository, NewApp)

	mainRouter := chi.NewRouter()
	urlRouter := urls.URLRouter(&ctx, NewApp.Cfg, urlService, appLogger)
	healthRouter := healthcheck.HealthCheckRouter(&ctx, DBCon, appLogger)
	mainRouter.Mount("/", urlRouter)
	mainRouter.Mount("/ping", healthRouter)

	appLogger.Info("Starting server",
		zap.String("server address", NewApp.Cfg.ServerAddress),
	)
	return http.ListenAndServe(NewApp.Cfg.ServerAddress, mainRouter)
}

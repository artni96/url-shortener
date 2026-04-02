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
	appLogger, err := logger.InitLogger(cfg.DebugLevel)
	app := &config.App{
		DB:     nil,
		Cfg:    cfg,
		Logger: appLogger,
	}

	ctx := context.Background()

	DBCon, err := db.InitDBConnection(ctx, app)
	defer DBCon.Close()

	if DBCon != nil {
		app.DB = DBCon
	}

	urlRepository, err := repository.NewURLRepository(app)
	if err != nil {
		return err
	}
	urlService := service.NewURLService(urlRepository, app)

	mainRouter := chi.NewRouter()
	urlRouter := urls.URLRouter(&ctx, app, urlService)
	healthRouter := healthcheck.HealthCheckRouter(&ctx, DBCon, appLogger)
	mainRouter.Mount("/", urlRouter)
	mainRouter.Mount("/ping", healthRouter)

	appLogger.Info("Starting server",
		zap.String("server address", app.Cfg.ServerAddress),
	)
	return http.ListenAndServe(app.Cfg.ServerAddress, mainRouter)
}

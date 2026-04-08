package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/config/db"
	"github.com/artni96/url-shortener/internal/handler/healthcheck"
	"github.com/artni96/url-shortener/internal/handler/urls"
	"github.com/artni96/url-shortener/internal/logger"
	urlrepo "github.com/artni96/url-shortener/internal/repository/urls"
	"github.com/artni96/url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func run(cfg *config.Config) error {
	ctx := context.Background()
	appLogger, err := logger.InitLogger(cfg.DebugLevel)

	app := &config.App{
		DB:     nil,
		Cfg:    cfg,
		Logger: appLogger,
	}

	DBCon, err := db.InitDBConnection(ctx, app)

	if err != nil {
		app.Logger.Info("failed to connect to database", zap.Error(err))
	}

	var urlDBRepository *urlrepo.DBURLRepository
	var urlInMemoryRepository *urlrepo.InMemoryURLRepository
	if DBCon != nil {
		app.DB = DBCon
		urlDBRepository, err = urlrepo.NewDBURLRepository(app)
		defer DBCon.Close()
	} else {
		urlInMemoryRepository, err = urlrepo.NewInMemoryURLRepository(app)
	}
	//if err != nil {
	//	return err
	//}

	//urlRepository, err := repository.NewURLRepository(app)
	if err != nil {
		app.Logger.Error("failed to initialize url repository", zap.Error(err))
		return fmt.Errorf("url repository is not initialized: %w", err)
	}
	urlService := service.NewURLService(urlDBRepository, urlInMemoryRepository, app)

	mainRouter := chi.NewRouter()
	urlRouter := urls.URLRouter(&ctx, app, urlService)
	healthRouter := healthcheck.HealthCheckRouter(&ctx, app)
	mainRouter.Mount("/", urlRouter)
	mainRouter.Mount("/ping", healthRouter)

	appLogger.Info("Starting server",
		zap.String("server address", app.Cfg.ServerAddress),
	)
	return http.ListenAndServe(app.Cfg.ServerAddress, mainRouter)
}

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
	authrepo "github.com/artni96/url-shortener/internal/repository/users"
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

	var userDBRepository *authrepo.DBUserRepository
	var userInMemoryRepository *authrepo.InMemoryUserRepository
	var userService *service.UserService

	var urlDBRepository *urlrepo.DBURLRepository
	var urlInMemoryRepository *urlrepo.InMemoryURLRepository
	var urlService *service.URLService

	if DBCon != nil {
		app.DB = DBCon
		urlDBRepository, err = urlrepo.NewDBURLRepository(app)
		if err != nil {
			app.Logger.Error("failed to initialize db url repository", zap.Error(err))
			return fmt.Errorf("url db repository is not initialized: %w", err)
		}
		urlService = service.NewURLService(urlDBRepository, nil, app)

		userDBRepository, err = authrepo.NewUserDBRepository(app)
		if err != nil {
			app.Logger.Error("failed to initialize users db repository", zap.Error(err))
			return fmt.Errorf("users db repository is not initialized: %w", err)
		}
		userService = service.NewUserService(userDBRepository, nil, app)
		defer DBCon.Close()
	} else {
		urlInMemoryRepository, err = urlrepo.NewInMemoryURLRepository(app)
		if err != nil {
			app.Logger.Error("failed to initialize url in-memory repository", zap.Error(err))
			return fmt.Errorf("url in-memory repository is not initialized: %w", err)
		}
		urlService = service.NewURLService(nil, urlInMemoryRepository, app)

		userInMemoryRepository, err = authrepo.NewInMemoryUserRepository(app)
		userService = service.NewUserService(nil, userInMemoryRepository, app)
	}

	mainRouter := chi.NewRouter()
	urlRouter := urls.URLRouter(&ctx, app, urlService, userService, cfg)
	healthRouter := healthcheck.HealthCheckRouter(&ctx, app)
	mainRouter.Mount("/", urlRouter)
	mainRouter.Mount("/ping", healthRouter)

	appLogger.Info("Starting server",
		zap.String("server address", app.Cfg.ServerAddress),
	)
	return http.ListenAndServe(app.Cfg.ServerAddress, mainRouter)
}

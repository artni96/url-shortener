package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/config/db"
	"github.com/artni96/url-shortener/internal/handler/auth"
	"github.com/artni96/url-shortener/internal/handler/healthcheck"
	"github.com/artni96/url-shortener/internal/handler/urls"
	"github.com/artni96/url-shortener/internal/logger"
	authrepo "github.com/artni96/url-shortener/internal/repository/auth"
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

	var authDBRepository *authrepo.DBAuthRepository
	var authInMemoryRepository *authrepo.InMemoryAuthRepository
	var authService *service.AuthService

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

		authDBRepository, err = authrepo.NewAuthDBRepository(app)
		if err != nil {
			app.Logger.Error("failed to initialize auth db repository", zap.Error(err))
			return fmt.Errorf("auth db repository is not initialized: %w", err)
		}
		authService = service.NewAuthService(authDBRepository, nil, app)
		defer DBCon.Close()
	} else {
		urlInMemoryRepository, err = urlrepo.NewInMemoryURLRepository(app)
		if err != nil {
			app.Logger.Error("failed to initialize url in-memory repository", zap.Error(err))
			return fmt.Errorf("url in-memory repository is not initialized: %w", err)
		}
		urlService = service.NewURLService(nil, urlInMemoryRepository, app)

		authInMemoryRepository, err = authrepo.NewInMemoryAuthRepository(app)
		authService = service.NewAuthService(nil, authInMemoryRepository, app)
	}

	mainRouter := chi.NewRouter()
	urlRouter := urls.URLRouter(&ctx, app, urlService)
	authRouter := auth.AuthRouter(&ctx, app, *authService)
	healthRouter := healthcheck.HealthCheckRouter(&ctx, app)
	mainRouter.Mount("/", urlRouter)
	mainRouter.Mount("/ping", healthRouter)
	mainRouter.Mount("/api/auth", authRouter)

	appLogger.Info("Starting server",
		zap.String("server address", app.Cfg.ServerAddress),
	)
	return http.ListenAndServe(app.Cfg.ServerAddress, mainRouter)
}

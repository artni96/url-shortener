package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/artni96/url-shortener/internal/audit"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"

	_ "github.com/artni96/url-shortener/api/docs"
	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/config/db"
	"github.com/artni96/url-shortener/internal/handler/healthcheck"
	"github.com/artni96/url-shortener/internal/handler/urls"
	"github.com/artni96/url-shortener/internal/logger"
	"github.com/artni96/url-shortener/internal/model"
	urlrepo "github.com/artni96/url-shortener/internal/repository/urls"
	authrepo "github.com/artni96/url-shortener/internal/repository/users"
	"github.com/artni96/url-shortener/internal/service"
)

func run(cfg *config.Config) error {

	ctx := context.Background()
	appLogger, err := logger.InitLogger(cfg.DebugLevel)
	if err != nil {
		log.Fatalf("failed to initialize application logger: %v", err)
		return err
	}
	auditChan := make(chan model.AuditEntity, 100)

	app := &config.App{
		DB:        nil,
		Cfg:       cfg,
		Logger:    appLogger,
		AuditChan: auditChan,
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

	mainRouter.Get("/swagger/*", httpSwagger.WrapHandler)

	urlRouter := urls.URLRouter(&ctx, app, urlService, userService, cfg)
	healthRouter := healthcheck.HealthCheckRouter(&ctx, app)
	mainRouter.Mount("/", urlRouter)
	mainRouter.Mount("/ping", healthRouter)

	mainRouter.HandleFunc("/debug/pprof/*", func(w http.ResponseWriter, r *http.Request) {
		http.DefaultServeMux.ServeHTTP(w, r)
	})

	appLogger.Info("Starting server",
		zap.String("server address", app.Cfg.ServerAddress),
	)

	newServer := &http.Server{
		Addr:    app.Cfg.ServerAddress,
		Handler: mainRouter,
	}

	go audit.RunAudit(app)

	go func() {
		err = newServer.ListenAndServe()
		if err != nil {
			app.Logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt)
	<-shutdownChan

	gsPeriod := time.Second * 5
	gsCtx, gsCancel := context.WithTimeout(ctx, gsPeriod)
	defer gsCancel()

	close(shutdownChan)
	close(auditChan)

	app.Logger.Info("shutting app down", zap.Time("time", time.Now()))
	go func() {
		deadline, _ := gsCtx.Deadline()
		for i := deadline.Second() - time.Now().Second(); i > 0; i-- {
			app.Logger.Info(fmt.Sprintf("app shutdown in %d sec", i))
			time.Sleep(1 * time.Second)
		}
	}()
	select {
	case <-gsCtx.Done():
		if app.DB != nil {
			if err = app.DB.Close(); err != nil {
				app.Logger.Info("failed to close database", zap.Error(err))
			} else if app.DB == nil {
				app.Logger.Info("database connection closed gracefully ")
			}
		}
	}

	if err = newServer.Shutdown(gsCtx); err != nil {
		app.Logger.Info("failed to shutdown server", zap.Error(err))
	} else {
		app.Logger.Info("server stopped gracefully")
	}
	app.Logger.Info("app stopped gracefully")

	return nil
}

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/artni96/url-shortener/api/docs"
	"github.com/artni96/url-shortener/internal/audit"
	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/config/db"
	"github.com/artni96/url-shortener/internal/handler/healthcheck"
	"github.com/artni96/url-shortener/internal/handler/urls"
	"github.com/artni96/url-shortener/internal/logger"
	"github.com/artni96/url-shortener/internal/model"
	urlrepo "github.com/artni96/url-shortener/internal/repository/urls"
	authrepo "github.com/artni96/url-shortener/internal/repository/users"
	"github.com/artni96/url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
)

type OutputConfig struct {
	Mode            string
	ServerAddress   string
	BaseURL         string
	FileStoragePath string
	EnableHTTPS     bool
	AuditFile       string
	AuditURL        string
	DebugLevel      string
	HostWhitelist   []string
}

func run(cfg *config.Config) error {

	ctx := context.Background()
	appLogger, err := logger.InitLogger(cfg.DebugLevel)
	if err != nil {
		log.Printf("failed to initialize application logger: %v\n", err)
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

	go audit.RunAudit(app)

	newServer := &http.Server{
		Addr:    app.Cfg.ServerAddress,
		Handler: mainRouter,
	}

	go func() error {
		err = runServer(app, newServer)
		if err != nil {

			app.Logger.Error("failed to start server", zap.Error(err))
			return err
		}
		return nil
	}()

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	<-shutdownChan

	gsPeriod := time.Second * 5
	gsCtx, gsCancel := context.WithTimeout(ctx, gsPeriod)
	defer gsCancel()

	close(shutdownChan)
	close(auditChan)

	app.Logger.Info("shutting the app down", zap.Time("time", time.Now()))
	if err = newServer.Shutdown(gsCtx); err != nil {
		app.Logger.Info("failed to shutdown server", zap.Error(err))
	} else {
		app.Logger.Info("server stopped gracefully, keep on processing left requests")
	}
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
	app.Logger.Info("app stopped gracefully")

	return nil
}

// stdoutConfig provides stdout of the app config.
func stdoutConfig(cfg *config.Config) (string, error) {
	outputConfig := &OutputConfig{
		Mode:            cfg.Mode,
		ServerAddress:   cfg.ServerAddress,
		BaseURL:         cfg.ResponseDomain,
		FileStoragePath: cfg.FileStoragePath,
		EnableHTTPS:     cfg.EnableHTTPS,
		AuditFile:       cfg.AuditFile,
		AuditURL:        cfg.AuditURL,
		DebugLevel:      cfg.DebugLevel,
		HostWhitelist:   cfg.HostWhitelist,
	}
	data, err := json.MarshalIndent(outputConfig, "", "  ")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s", data), nil
}

// runServer launches the server according to the app config.
func runServer(app *config.App, server *http.Server) error {
	outputConfig, err := stdoutConfig(app.Cfg)
	if err != nil {
		app.Logger.Error("failed to prepare output config data", zap.Error(err))
		return err
	}

	if app.Cfg.EnableHTTPS {
		if app.Cfg.Mode == "prod" {
			manager := &autocert.Manager{
				Cache:      autocert.DirCache("cache"),
				Prompt:     autocert.AcceptTOS,
				HostPolicy: autocert.HostWhitelist(app.Cfg.HostWhitelist...),
			}
			server.TLSConfig = manager.TLSConfig()
		}

		app.Logger.Info(outputConfig)
		err = server.ListenAndServeTLS(app.Cfg.CertFile, app.Cfg.KeyFile)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.Logger.Error("failed to start HTTPS server", zap.Error(err))
			return err
		}
		return nil
	}

	app.Logger.Info(outputConfig)
	err = server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		app.Logger.Error("failed to start HTTP server", zap.Error(err))
		return err
	}
	return nil
}

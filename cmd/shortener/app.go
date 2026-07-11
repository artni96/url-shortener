package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
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
	"golang.org/x/sync/errgroup"
)

// run launches the app with all dependencies.
func run(cfg *config.Config) error {
	ctx := context.Background()
	eg := new(errgroup.Group)
	appLogger, err := logger.InitLogger(cfg.DebugLevel)
	if err != nil {
		log.Printf("failed to initialize application logger: %v\n", err)
		return err
	}
	auditChan := make(chan model.AuditEntity, 100)
	isClosedChan := make(chan bool, 1)

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

	shutdownCtx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	defer stop()

	var errRunAudit error
	eg.Go(func() error {
		if err = audit.RunAudit(app); err != nil {
			if errRunAudit = shutdownCtx.Err(); errRunAudit != nil {
				app.Logger.Error("a closing signal received, failed to maintain running audit service", zap.Error(err))
				return errRunAudit
			}
			return err
		}
		return nil
	})

	newServer := &http.Server{
		Addr:    app.Cfg.ServerAddress,
		Handler: mainRouter,
	}
	var errRunServer error
	eg.Go(func() error {
		if err = runServer(app, newServer); err != nil {
			if errRunServer = shutdownCtx.Err(); errRunServer != nil {
				app.Logger.Info("a closing signal received, failed to maintain running server", zap.Error(err))
				return errRunServer
			}
			return err
		}
		return nil
	})

	<-shutdownCtx.Done()

	gsPeriod := time.Second * 5
	gsCtx, gsCancel := context.WithTimeout(ctx, gsPeriod)
	defer gsCancel()

	app.Logger.Info("shutting the app down", zap.Time("time", time.Now()))

	eg.Go(func() error {
		select {
		case <-gsCtx.Done():
			if len(isClosedChan) == 0 {
				app.Logger.Info("graceful period is out!")
			}
			fsCtx, fsCancel := context.WithTimeout(ctx, time.Second*30)
			defer fsCancel()
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			deadline, _ := fsCtx.Deadline()
			for {
				timeLeft := int(math.Ceil(time.Until(deadline).Seconds()))
				if timeLeft > 0 {
					select {
					case <-isClosedChan:
						close(isClosedChan)
						return nil
					default:
						<-ticker.C
						app.Logger.Info(fmt.Sprintf("forceful app shutdown in %d sec", timeLeft-1))
					}
				} else if timeLeft == 0 {
					app.Logger.Info(fmt.Sprintf("app stopped forcefully"))
					return nil
				}
			}
		}
	})

	eg.Go(func() error {
		if err = newServer.Shutdown(gsCtx); err != nil {
			app.Logger.Info("failed to shutdown server gracefully", zap.Error(err))
		} else {
			close(auditChan)
			app.Logger.Info("server stopped gracefully, keep on processing left requests")
			if app.DB != nil {
				if err = app.DB.Close(); err != nil {
					app.Logger.Info("failed to close database gracefully", zap.Error(err))
				} else {
					isClosedChan <- true
					gsCancel()
					app.Logger.Info("database connection closed gracefully ")
				}
			}
		}
		return nil
	})

	if err = eg.Wait(); err != nil {
		app.Logger.Info("failed to wait for errgroup goroutines completion", zap.Error(err))
		return err
	}

	app.Logger.Info("app stopped gracefully")
	return nil
}

// stdoutConfig provides stdout of the app config.
func stdoutConfig(cfg *config.Config) (string, error) {
	var resp string
	resp += fmt.Sprintf("\n\nApp config: \n")
	resp += fmt.Sprintf("	Mode: %s\n", cfg.Mode)
	resp += fmt.Sprintf("	Server Address: %s\n", cfg.ServerAddress)
	resp += fmt.Sprintf("	Base URL: %s\n", cfg.ResponseDomain)
	if cfg.DatabaseDsn != "" {
		resp += fmt.Sprintf("	Database is applied: %t\n", true)
	} else {
		resp += fmt.Sprintf("	Database is applied: %t\n", false)
	}
	if cfg.FileStoragePath != "" {
		resp += fmt.Sprintf("	Storage Path is applied: %t\n", true)
	} else {
		resp += fmt.Sprintf("	Storage Path is applied: %t\n", false)
	}
	resp += fmt.Sprintf("	HTTPS is enable: %t\n", cfg.EnableHTTPS)
	if cfg.AuditFile != "" {
		resp += fmt.Sprintf("	Audit File is applied: %t\n", true)
	} else {
		resp += fmt.Sprintf("	Audit File is applied: %t\n", false)
	}
	if cfg.AuditURL != "" {
		resp += fmt.Sprintf("	Audit URL is applied: %t\n", true)
	} else {
		resp += fmt.Sprintf("	Audit URL is applied: %t\n", false)
	}
	resp += fmt.Sprintf("	Debug Level: %s\n", cfg.DebugLevel)
	if cfg.CertFile != "" {
		resp += fmt.Sprintf("	Certificate is applied: %t\n", true)
	} else {
		resp += fmt.Sprintf("	Certificate is applied: %t\n", false)
	}
	if cfg.KeyFile != "" {
		resp += fmt.Sprintf("	Key file is applied: %t\n", true)
	} else {
		resp += fmt.Sprintf("	Key file is applied: %t\n", false)
	}
	resp += fmt.Sprintf("	Host white list: %s", cfg.HostWhitelist)

	return resp, nil
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

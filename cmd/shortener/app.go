package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/artni96/url-shortener/api/docs"
	"github.com/artni96/url-shortener/cmd/shortener/servers"
	"github.com/artni96/url-shortener/internal/audit"
	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/config/db"
	"github.com/artni96/url-shortener/internal/handler/grpc/interceptors"
	"github.com/artni96/url-shortener/internal/handler/http/healthcheck"
	statshandler "github.com/artni96/url-shortener/internal/handler/http/stats"
	"github.com/artni96/url-shortener/internal/handler/http/urls"
	"github.com/artni96/url-shortener/internal/logger"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/repository/stats"
	urlrepo "github.com/artni96/url-shortener/internal/repository/urls"
	authrepo "github.com/artni96/url-shortener/internal/repository/users"
	"github.com/artni96/url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
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
	isClosedChan := make(chan struct{})

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

	var statsDBRepository *stats.DBStatsRepository
	var statsService *service.StatsService

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

		statsDBRepository, err = stats.NewDBStatsRepository(app)
		if err != nil {
			app.Logger.Error("failed to initialize stats db repository", zap.Error(err))
			return fmt.Errorf("stats db repository is not initialized: %w", err)
		}
		statsService = service.NewStatsService(statsDBRepository, app, userService, urlService)

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

		statsService = service.NewStatsService(nil, app, userService, urlService)
	}

	mainRouter := chi.NewRouter()

	mainRouter.Get("/swagger/*", httpSwagger.WrapHandler)

	urlRouter := urls.URLRouter(&ctx, app, urlService, userService, cfg)
	mainRouter.Mount("/", urlRouter)

	healthRouter := healthcheck.HealthCheckRouter(&ctx, app)
	mainRouter.Mount("/ping", healthRouter)

	statsRouter := statshandler.StatsRouter(ctx, app, statsService, cfg.TrustedSubnet)
	mainRouter.Mount("/api/internal", statsRouter)

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

	newHTTPServer := &http.Server{
		Addr:    app.Cfg.ServerAddress,
		Handler: mainRouter,
	}

	newGRPCServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.AuthInterceptor(app),
			interceptors.SubnetInterceptor(app)),
	)

	var errRunServer error

	outputConfig, err := stdoutConfig(app.Cfg)
	if err != nil {
		app.Logger.Error("failed to prepare output config data", zap.Error(err))
		return err
	}
	app.Logger.Info(outputConfig)

	eg.Go(func() error {
		if err = servers.NewHTTPServer(app, newHTTPServer); err != nil {
			if errRunServer = shutdownCtx.Err(); errRunServer != nil {
				app.Logger.Info("a closing signal received, failed to maintain running server", zap.Error(err))
				return errRunServer
			}
			return err
		}
		return nil
	})

	eg.Go(func() error {
		err = servers.NewGRPCServer(newGRPCServer, app, *urlService, *userService)
		if errRunServer = shutdownCtx.Err(); errRunServer != nil {
			select {
			case _, ok := <-isClosedChan:
				if ok {
					app.Logger.Info("failed to launch GRPC server", zap.Error(err))
				} else {
					app.Logger.Info("gRPC server stopped gracefully")
				}
			}

		}
		return nil
	})

	<-shutdownCtx.Done()

	gsPeriod := time.Second * 5
	gsCtx, gsCancel := context.WithTimeout(ctx, gsPeriod)
	defer gsCancel()

	app.Logger.Info("shutting the app down", zap.Time("time", time.Now()))

	eg.Go(func() error {
		<-gsCtx.Done()
		select {
		case <-isClosedChan:
			return nil
		default:
			app.Logger.Info("graceful period has expired", zap.Time("time", time.Now()))
			app.Logger.Info("app will be shutdown forcefully in 30 seconds")
			fsCtx, fsCancel := context.WithTimeout(ctx, time.Second*30)
			defer fsCancel()

			go fsCountdown(fsCtx, app, isClosedChan)

			select {
			case <-fsCtx.Done():
				app.Logger.Info("app stopped forcefully", zap.Time("time", time.Now()))
				os.Exit(0)
			case _, ok := <-isClosedChan:
				if !ok {
					fsCancel()
				}
			}
		}
		return nil
	})

	eg.Go(func() error {
		if err = newHTTPServer.Shutdown(gsCtx); err != nil {
			app.Logger.Info("failed to shutdown server gracefully", zap.Error(err))
			return err
		}
		newGRPCServer.GracefulStop()
		close(auditChan)
		app.Logger.Info("HTTP server stopped gracefully, keep on processing left requests")
		if app.DB != nil {
			if err = app.DB.Close(); err != nil {
				app.Logger.Info("failed to close database gracefully", zap.Error(err))
				return err
			}
		}
		close(isClosedChan)
		gsCancel()
		app.Logger.Info("database connection closed gracefully")
		return nil
	})

	if err = eg.Wait(); err != nil {
		app.Logger.Info("failed to wait for errgroup goroutines completion", zap.Error(err))
		return err
	}

	app.Logger.Info("app stopped gracefully")
	return nil
}

// fsCountdown counts down left time of forceful shutdown.
func fsCountdown(ctx context.Context, app *config.App, isClosedChan <-chan struct{}) {
	deadline, _ := ctx.Deadline()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-isClosedChan:
			return
		case <-ticker.C:
			timeLeft := int(math.Ceil(time.Until(deadline).Seconds()))
			if timeLeft == 0 {
				return
			}
			if timeLeft == 15 || timeLeft == 10 || (timeLeft <= 5 && timeLeft > 0) {
				app.Logger.Info(fmt.Sprintf("forceful app shutdown in %d sec", timeLeft))
			}
		}
	}
}

// stdoutConfig provides stdout of the app config.
func stdoutConfig(cfg *config.Config) (string, error) {
	var resp string
	resp += fmt.Sprintf("\n\nApp config: \n")
	resp += fmt.Sprintf("	Mode: %s\n", cfg.Mode)
	resp += fmt.Sprintf("	HTTP Server Address: %s\n", cfg.ServerAddress)
	resp += fmt.Sprintf("	gRPC Server Address: %s\n", cfg.GRPCAddress)
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

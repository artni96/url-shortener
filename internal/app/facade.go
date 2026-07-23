package app

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/artni96/url-shortener/cmd/shortener/servers"
	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/config/db"
	"github.com/artni96/url-shortener/internal/handler/http/healthcheck"
	statshandler "github.com/artni96/url-shortener/internal/handler/http/stats"
	"github.com/artni96/url-shortener/internal/handler/http/urls"
	applogger "github.com/artni96/url-shortener/internal/logger"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/repository/stats"
	urlrepo "github.com/artni96/url-shortener/internal/repository/urls"
	authrepo "github.com/artni96/url-shortener/internal/repository/users"
	"github.com/artni96/url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// AppFacade is the facade for the app with all dependencies.
type AppFacade struct {
	eg     *errgroup.Group
	Cfg    *config.Config
	DB     *sqlx.DB
	Logger *zap.Logger

	httpServer *servers.HTTPServer
	httpRouter *chi.Mux

	grpcServer *servers.GRPCServer

	urlDBRepository       *urlrepo.DBURLRepository
	urlInMemoryRepository *urlrepo.InMemoryURLRepository
	urlService            *service.URLService

	userDBRepository       *authrepo.DBUserRepository
	userInMemoryRepository *authrepo.InMemoryUserRepository
	userService            *service.UserService

	statsDBRepository *stats.DBStatsRepository
	statsService      *service.StatsService

	AuditChan    chan model.AuditEntity
	isClosedChan chan struct{}
}

// initLogger initializes the app logger.
func (f *AppFacade) initLogger() error {
	logger, err := applogger.InitLogger(f.Cfg.DebugLevel)
	if err != nil {
		log.Printf("failed to initialize app logger: %v\n", err)
		return fmt.Errorf("failed to initialize app logger: %w", err)
	}
	f.Logger = logger
	return nil
}

// InitServers initializes the app HTTP and gRPC servers.
func (f *AppFacade) InitServers(ctx context.Context) {
	f.initHTTPRouter(ctx)
	f.httpServer = servers.NewHTTPServer(f.Cfg, f.Logger, f.httpRouter)
	f.grpcServer = servers.NewGRPCServer(f.Cfg, f.Logger, f.urlService, f.userService)
}

func (f *AppFacade) initHTTPRouter(ctx context.Context) {
	httpRouter := chi.NewRouter()

	httpRouter.Get("/swagger/*", httpSwagger.WrapHandler)

	urlRouter := urls.URLRouter(&ctx, f.Logger, f.urlService, f.userService, f.Cfg, f.AuditChan)
	httpRouter.Mount("/", urlRouter)

	healthRouter := healthcheck.HealthCheckRouter(&ctx, f.Logger, f.DB)
	httpRouter.Mount("/ping", healthRouter)

	statsRouter := statshandler.StatsRouter(f.Logger, f.statsService, f.Cfg.TrustedSubnet)
	httpRouter.Mount("/api/internal", statsRouter)

	httpRouter.HandleFunc("/debug/pprof/*", func(w http.ResponseWriter, r *http.Request) {
		http.DefaultServeMux.ServeHTTP(w, r)
	})
	f.httpRouter = httpRouter
}

// initDBConn initializes connection to a database according to the app config.
func (f *AppFacade) initDBConn(ctx context.Context) {
	DBConn, err := db.InitDBConnection(ctx, f.Cfg, f.Logger)
	if err != nil {
		f.Logger.Error("failed to connect to database", zap.Error(err))
	}
	f.DB = DBConn
}

// InitDependencies initializes all app servers and repositories.
func (f *AppFacade) InitDependencies(ctx context.Context) error {
	err := f.initLogger()
	if err != nil {
		return err
	}
	f.initDBConn(ctx)

	if f.DB != nil {
		urlDBRepository, err := urlrepo.NewDBURLRepository(f.DB, f.Logger)
		if err != nil {
			f.Logger.Error("failed to initialize db url repository", zap.Error(err))
			return fmt.Errorf("url db repository is not initialized: %w", err)
		}
		f.urlDBRepository = urlDBRepository
		f.urlService = service.NewURLService(urlDBRepository, nil, f.Cfg, f.Logger)

		userDBRepository, err := authrepo.NewUserDBRepository(f.DB, f.Logger)
		if err != nil {
			f.Logger.Error("failed to initialize users db repository", zap.Error(err))
			return fmt.Errorf("users db repository is not initialized: %w", err)
		}
		f.userDBRepository = userDBRepository
		f.userService = service.NewUserService(userDBRepository, nil, f.Cfg, f.Logger)

		statsDBRepository, err := stats.NewDBStatsRepository(f.DB, f.Logger)
		if err != nil {
			f.Logger.Error("failed to initialize stats db repository", zap.Error(err))
			return fmt.Errorf("stats db repository is not initialized: %w", err)
		}
		f.statsDBRepository = statsDBRepository
		f.statsService = service.NewStatsService(statsDBRepository, f.DB, f.userService, f.urlService)

	} else {
		urlInMemoryRepository, err := urlrepo.NewInMemoryURLRepository(f.Cfg, f.Logger)
		if err != nil {
			f.Logger.Error("failed to initialize url in-memory repository", zap.Error(err))
			return fmt.Errorf("url in-memory repository is not initialized: %w", err)
		}
		f.urlInMemoryRepository = urlInMemoryRepository
		f.urlService = service.NewURLService(nil, urlInMemoryRepository, f.Cfg, f.Logger)

		userInMemoryRepository, err := authrepo.NewInMemoryUserRepository(f.Cfg, f.Logger)
		if err != nil {
			f.Logger.Error("failed to initialize users in-memory repository", zap.Error(err))
			return fmt.Errorf("users in-memory repository is not initialized: %w", err)
		}
		f.userInMemoryRepository = userInMemoryRepository
		f.userService = service.NewUserService(nil, userInMemoryRepository, f.Cfg, f.Logger)

		f.statsService = service.NewStatsService(nil, f.DB, f.userService, f.urlService)
	}
	return nil
}

// CloseDB closes database connection.
func (f *AppFacade) CloseDB() {
	f.DB.Close()
}

// StartServers launches the app HTTP and gRPC servers.
func (f *AppFacade) StartServers(ctx context.Context) {
	f.InitServers(ctx)

	f.eg.Go(func() error {
		if err := f.httpServer.RunHTTPServer(); err != nil {
			if errRunServer := ctx.Err(); errRunServer != nil {
				f.Logger.Info("a closing signal received, failed to maintain running server", zap.Error(err))
				return errRunServer
			}
			return err
		}
		return nil
	})

	f.eg.Go(func() error {
		err := f.grpcServer.RunGRPCServer()
		if errRunServer := ctx.Err(); errRunServer != nil {
			select {
			case _, ok := <-f.isClosedChan:
				if ok {
					f.Logger.Info("failed to launch GRPC server", zap.Error(err))
				} else {
					f.Logger.Info("gRPC server stopped gracefully")
				}
			}
		}
		return nil
	})
}

// StopServers stops the app HTTP and gRPC servers.
func (f *AppFacade) StopServers(ctx context.Context, cancel context.CancelFunc) {
	f.eg.Go(func() error {
		if err := f.httpServer.Shutdown(ctx); err != nil {
			f.Logger.Info("failed to shutdown server gracefully", zap.Error(err))
			return err
		}

		f.grpcServer.Shutdown()
		close(f.AuditChan)
		f.Logger.Info("HTTP server stopped gracefully, keep on processing left requests")
		if f.DB != nil {
			if err := f.DB.Close(); err != nil {
				f.Logger.Info("failed to close database gracefully", zap.Error(err))
				return err
			}
		}
		close(f.isClosedChan)
		cancel()
		f.Logger.Info("database connection closed gracefully")
		return nil
	})
}

// NewAppFacade initializes and returns a new instance of AppFacade.
func NewAppFacade(eg *errgroup.Group, cfg *config.Config, auditChan chan model.AuditEntity, isClosedChan chan struct{}) *AppFacade {
	return &AppFacade{
		eg:  eg,
		Cfg: cfg,

		AuditChan:    auditChan,
		isClosedChan: isClosedChan,
	}
}

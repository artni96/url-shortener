package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/artni96/url-shortener/cmd/shortener/servers"
	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/config/db"
	"github.com/artni96/url-shortener/internal/handler/http/healthcheck"
	statshandler "github.com/artni96/url-shortener/internal/handler/http/stats"
	"github.com/artni96/url-shortener/internal/handler/http/urls"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/repository/stats"
	urlrepo "github.com/artni96/url-shortener/internal/repository/urls"
	authrepo "github.com/artni96/url-shortener/internal/repository/users"
	"github.com/artni96/url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type AppFacade struct {
	eg  *errgroup.Group
	app *config.App

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

	auditChan    chan model.AuditEntity
	isClosedChan chan struct{}
}

func (f *AppFacade) InitServers(ctx context.Context) {
	f.initHTTPRouter(ctx)
	f.httpServer = servers.NewHTTPServer(f.app, f.httpRouter)
	f.grpcServer = servers.NewGRPCServer(f.app, f.urlService, f.userService)
}

func (f *AppFacade) initHTTPRouter(ctx context.Context) {
	httpRouter := chi.NewRouter()

	httpRouter.Get("/swagger/*", httpSwagger.WrapHandler)

	urlRouter := urls.URLRouter(&ctx, f.app, f.urlService, f.userService, f.app.Cfg)
	httpRouter.Mount("/", urlRouter)

	healthRouter := healthcheck.HealthCheckRouter(&ctx, f.app)
	httpRouter.Mount("/ping", healthRouter)

	statsRouter := statshandler.StatsRouter(f.app, f.statsService, f.app.Cfg.TrustedSubnet)
	httpRouter.Mount("/api/internal", statsRouter)

	httpRouter.HandleFunc("/debug/pprof/*", func(w http.ResponseWriter, r *http.Request) {
		http.DefaultServeMux.ServeHTTP(w, r)
	})
	f.httpRouter = httpRouter
}

func (f *AppFacade) initDBConn(ctx context.Context) {
	DBConn, err := db.InitDBConnection(ctx, f.app)
	if err != nil {
		f.app.Logger.Error("failed to connect to database", zap.Error(err))
	}
	f.app.DB = DBConn
}

func (f *AppFacade) InitDependencies(ctx context.Context) error {
	f.initDBConn(ctx)

	if f.app.DB != nil {
		urlDBRepository, err := urlrepo.NewDBURLRepository(f.app)
		if err != nil {
			f.app.Logger.Error("failed to initialize db url repository", zap.Error(err))
			return fmt.Errorf("url db repository is not initialized: %w", err)
		}
		f.urlDBRepository = urlDBRepository
		f.urlService = service.NewURLService(urlDBRepository, nil, f.app)

		userDBRepository, err := authrepo.NewUserDBRepository(f.app)
		if err != nil {
			f.app.Logger.Error("failed to initialize users db repository", zap.Error(err))
			return fmt.Errorf("users db repository is not initialized: %w", err)
		}
		f.userDBRepository = userDBRepository
		f.userService = service.NewUserService(userDBRepository, nil, f.app)

		statsDBRepository, err := stats.NewDBStatsRepository(f.app)
		if err != nil {
			f.app.Logger.Error("failed to initialize stats db repository", zap.Error(err))
			return fmt.Errorf("stats db repository is not initialized: %w", err)
		}
		f.statsDBRepository = statsDBRepository
		f.statsService = service.NewStatsService(statsDBRepository, f.app, f.userService, f.urlService)

	} else {
		urlInMemoryRepository, err := urlrepo.NewInMemoryURLRepository(f.app)
		if err != nil {
			f.app.Logger.Error("failed to initialize url in-memory repository", zap.Error(err))
			return fmt.Errorf("url in-memory repository is not initialized: %w", err)
		}
		f.urlInMemoryRepository = urlInMemoryRepository
		f.urlService = service.NewURLService(nil, urlInMemoryRepository, f.app)

		userInMemoryRepository, err := authrepo.NewInMemoryUserRepository(f.app)
		if err != nil {
			f.app.Logger.Error("failed to initialize users in-memory repository", zap.Error(err))
			return fmt.Errorf("users in-memory repository is not initialized: %w", err)
		}
		f.userInMemoryRepository = userInMemoryRepository
		f.userService = service.NewUserService(nil, userInMemoryRepository, f.app)

		f.statsService = service.NewStatsService(nil, f.app, f.userService, f.urlService)
	}
	return nil
}

func (f *AppFacade) CloseDB() {
	f.app.DB.Close()
}

func (f *AppFacade) StartServers(ctx context.Context) {
	f.InitServers(ctx)

	f.eg.Go(func() error {
		if err := f.httpServer.RunHTTPServer(); err != nil {
			if errRunServer := ctx.Err(); errRunServer != nil {
				f.app.Logger.Info("a closing signal received, failed to maintain running server", zap.Error(err))
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
					f.app.Logger.Info("failed to launch GRPC server", zap.Error(err))
				} else {
					f.app.Logger.Info("gRPC server stopped gracefully")
				}
			}
		}
		return nil
	})
}

func (f *AppFacade) StopServers(ctx context.Context, cancel context.CancelFunc) {
	f.eg.Go(func() error {
		if err := f.httpServer.Shutdown(ctx); err != nil {
			f.app.Logger.Info("failed to shutdown server gracefully", zap.Error(err))
			return err
		}

		f.grpcServer.Shutdown()
		close(f.auditChan)
		f.app.Logger.Info("HTTP server stopped gracefully, keep on processing left requests")
		if f.app.DB != nil {
			if err := f.app.DB.Close(); err != nil {
				f.app.Logger.Info("failed to close database gracefully", zap.Error(err))
				return err
			}
		}
		close(f.isClosedChan)
		cancel()
		f.app.Logger.Info("database connection closed gracefully")
		return nil
	})
}

func NewAppFacade(eg *errgroup.Group, app *config.App, auditChan chan model.AuditEntity, isClosedChan chan struct{}) *AppFacade {
	return &AppFacade{
		eg:  eg,
		app: app,

		auditChan:    auditChan,
		isClosedChan: isClosedChan,
	}
}

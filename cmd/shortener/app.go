package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/artni96/url-shortener/api/docs"
	appFacade "github.com/artni96/url-shortener/internal/app"
	"github.com/artni96/url-shortener/internal/audit"
	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// run launches the app with all dependencies.
func run(cfg *config.Config) error {
	ctx := context.Background()
	eg := new(errgroup.Group)
	//appLogger, err := logger.InitLogger(cfg.DebugLevel)
	//if err != nil {
	//	log.Printf("failed to initialize application logger: %v\n", err)
	//	return err
	//}
	auditChan := make(chan model.AuditEntity, 100)
	isClosedChan := make(chan struct{})

	//appCfg := &config.App{
	//	DB:        nil,
	//	Cfg:       cfg,
	//	Logger:    appLogger,
	//	AuditChan: auditChan,
	//}

	app := appFacade.NewAppFacade(eg, cfg, auditChan, isClosedChan)
	err := app.InitDependencies(ctx)
	if err != nil {
		app.Logger.Info("failed to initialize app dependencies", zap.Error(err))
		return err
	}
	defer app.CloseDB()

	shutdownCtx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	defer stop()

	var errRunAudit error
	eg.Go(func() error {
		if err = audit.RunAudit(app.Cfg, app.Logger, app.AuditChan); err != nil {
			if errRunAudit = shutdownCtx.Err(); errRunAudit != nil {
				app.Logger.Error("a closing signal received, failed to maintain running audit service", zap.Error(err))
				return errRunAudit
			}
			return err
		}
		return nil
	})

	app.StartServers(shutdownCtx)

	outputConfig, err := stdoutConfig(app.Cfg)
	if err != nil {
		app.Logger.Error("failed to prepare output config data", zap.Error(err))
		return err
	}
	app.Logger.Info(outputConfig)

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

			go fsCountdown(fsCtx, app.Logger, isClosedChan)

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

	app.StopServers(gsCtx, gsCancel)

	if err = eg.Wait(); err != nil {
		app.Logger.Info("failed to wait for errgroup goroutines completion", zap.Error(err))
		return err
	}

	app.Logger.Info("app stopped gracefully")
	return nil
}

// fsCountdown counts down left time of forceful shutdown.
func fsCountdown(ctx context.Context, logger *zap.Logger, isClosedChan <-chan struct{}) {
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
				logger.Info(fmt.Sprintf("forceful app shutdown in %d sec", timeLeft))
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

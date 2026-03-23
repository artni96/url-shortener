package main

import (
	"log"
	"net/http"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/handler/urls"
	"github.com/artni96/url-shortener/internal/logger"
	"github.com/artni96/url-shortener/internal/repository"
	"github.com/artni96/url-shortener/internal/service"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.ParseFlags()
	if err != nil {
		log.Fatal(err.Error())
	}
	if err = run(cfg); err != nil {
		log.Fatal(err.Error())
	}
}

func run(cfg *config.Config) error {
	appLogger, err := logger.InitLogger(cfg.DebugLevel)
	if err != nil {
		return err
	}
	urlRepository, err := repository.NewURLRepository(cfg, appLogger)
	if err != nil {
		return err
	}
	urlService := service.NewURLService(urlRepository, cfg)
	urlHandler := urls.URLRouter(cfg, urlService, appLogger)

	appLogger.Info("Starting server",
		zap.String("server address", cfg.ServerAddress),
	)
	return http.ListenAndServe(cfg.ServerAddress, urlHandler)
}

package main

import (
	"net/http"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/handler/urls"
	"github.com/artni96/url-shortener/internal/logger"
	"github.com/artni96/url-shortener/internal/repository"
	"github.com/artni96/url-shortener/internal/service"
)

func main() {
	cfg, err := config.ParseFlags()
	if err != nil {
		logger.Logger.Fatal(err.Error())
	}
	if err = run(cfg); err != nil {
		logger.Logger.Fatal(err.Error())
	}
}

func run(cfg *config.Config) error {

	urlRepository := repository.NewURLRepository()
	urlService := service.NewURLService(urlRepository)
	urlHandler := urls.URLRouter(cfg, urlService)
	if err := logger.InitLogger(cfg.DebugLevel); err != nil {
		return err
	}
	logger.Logger.Infof("Starting server at address %s", cfg.ServerAddress)
	return http.ListenAndServe(cfg.ServerAddress, urlHandler)
}

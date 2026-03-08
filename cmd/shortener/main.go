package main

import (
	"log"
	"net/http"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/handler/urls"
	"github.com/artni96/url-shortener/internal/repository"
	"github.com/artni96/url-shortener/internal/service"
)

func main() {

	cfg, err := config.ParseFlags()
	urlRepository := repository.NewURLRepository()
	urlService := service.NewURLService(urlRepository)
	urlHandler := urls.URLRouter(cfg, urlService)
	if err != nil {
		log.Fatal(err)
	}
	err = http.ListenAndServe(cfg.ServerAddress, urlHandler)
	if err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"log"
	"net/http"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/handler/urls"
)

func main() {
	cfg, err := config.ParseFlags()
	if err != nil {
		log.Fatal(err)
	}
	err = http.ListenAndServe(cfg.ServerAddress, urls.URLRouter(cfg))
	if err != nil {
		log.Fatal(err)
	}
}

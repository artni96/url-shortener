package main

import (
	"net/http"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/handler/urls"
)

func main() {
	cfg, err := config.ParseFlags()
	if err != nil {
		panic(err)
	}
	err = http.ListenAndServe(cfg.MainDomain, urls.URLRouter(cfg))
	if err != nil {
		panic(err)
	}
}

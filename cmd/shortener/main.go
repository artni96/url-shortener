// Сервис url-shortener - это инструмент, который превращает громоздкий веб-адрес (URL) в короткую и удобную ссылку.

package main

import (
	"log"

	"github.com/artni96/url-shortener/internal/config"
)

// @title URL-shortener
// @version 1.0
// @description URL-shortener is a tool to shorten a long link and create a short URL easy to share on sites, chat and emails.
// @host localhost:8080
// BasePath /
func main() {
	cfg, err := config.ParseFlags()
	if err != nil {
		log.Fatal(err.Error())
	}
	if err = run(cfg); err != nil {
		log.Fatal(err.Error())
	}
}

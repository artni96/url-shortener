// Сервис url-shortener - это инструмент, который превращает громоздкий веб-адрес (URL) в короткую и удобную ссылку.

package main

import (
	"log"

	"github.com/artni96/url-shortener/internal/config"
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

package config

import (
	"flag"
)

type Config struct {
	MainDomain  string
	ResponseURL string
}

func ParseFlags() (*Config, error) {
	aFlag := flag.String("a", "", "request url")
	bFlag := flag.String("b", "", "response url")
	flag.Parse()
	mainDomain := "localhost:8080"
	if *aFlag != "" {
		mainDomain = *aFlag
	}
	responseURL := "http://localhost:8080"
	if *bFlag != "" {
		responseURL = *bFlag
	}

	conf := Config{
		MainDomain:  mainDomain,
		ResponseURL: responseURL,
	}
	return &conf, nil
}

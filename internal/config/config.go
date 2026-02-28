package config

import (
	"flag"
)

type Config struct {
	ServerAddress string
	ResponseURL   string
}

func ParseFlags() (*Config, error) {
	aFlag := flag.String("a", "", "request domain")
	bFlag := flag.String("b", "", "response url")
	flag.Parse()
	serverAddress := "localhost:8080"
	if *aFlag != "" {
		serverAddress = *aFlag
	}
	responseURL := "http://localhost:8080"
	if *bFlag != "" {
		responseURL = *bFlag
	}

	conf := Config{
		ServerAddress: serverAddress,
		ResponseURL:   responseURL,
	}
	return &conf, nil
}

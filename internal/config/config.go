package config

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddress string `env:"SERVER_ADDRESS"`
	ResponseURL   string `env:"BASE_URL"`
}

func ParseFlags() (*Config, error) {
	fs := flag.NewFlagSet("fs", flag.ExitOnError)
	conf := Config{}
	if envServerAddress := os.Getenv("SERVER_ADDRESS"); envServerAddress != "" {
		conf.ServerAddress = envServerAddress
	} else {
		fs.StringVar(&conf.ServerAddress, "a", "localhost:8080", "server address")
	}
	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		conf.ResponseURL = envBaseURL
	} else {
		fs.StringVar(&conf.ResponseURL, "b", "http://localhost:8080", "response URL")
	}
	err := fs.Parse(os.Args[1:])
	if err != nil {
		return nil, err
	}
	return &conf, nil
}

package config

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddress string `env:"SERVER_ADDRESS"`
	ResponseURL   string `env:"BASE_URL"`
	DebugLevel    string `env:"DEBUG_LEVEL"`
}

func ParseFlags() (*Config, error) {
	fs := flag.NewFlagSet("fs", flag.ExitOnError)
	conf := Config{}
	fs.StringVar(&conf.ServerAddress, "a", "localhost:8080", "server address")
	fs.StringVar(&conf.ResponseURL, "b", "http://localhost:8080", "response URL")
	fs.StringVar(&conf.DebugLevel, "debug", "Info", "debug level")

	err := fs.Parse(os.Args[1:])
	if err != nil {
		return nil, err
	}

	if envServerAddress := os.Getenv("SERVER_ADDRESS"); envServerAddress != "" {
		conf.ServerAddress = envServerAddress
	}
	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		conf.ResponseURL = envBaseURL
	}
	if envDebugLevel := os.Getenv("DEBUG_LEVEL"); envDebugLevel != "" {
		conf.DebugLevel = envDebugLevel
	}
	return &conf, nil
}

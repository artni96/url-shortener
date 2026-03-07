package config

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddress string
	ResponseURL   string
}

func ParseFlags() (*Config, error) {
	fs := flag.NewFlagSet("fs", flag.ExitOnError)
	conf := Config{}
	fs.StringVar(&conf.ServerAddress, "a", "localhost:8080", "server address")
	fs.StringVar(&conf.ResponseURL, "b", "http://localhost:8080", "response URL")
	err := fs.Parse(os.Args[1:])
	if err != nil {
		return nil, err
	}
	return &conf, nil
}

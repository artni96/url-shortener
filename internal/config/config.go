package config

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddress   string `env:"SERVER_ADDRESS"`
	ResponseURL     string `env:"BASE_URL"`
	DebugLevel      string `env:"DEBUG_LEVEL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
}

func ParseFlags() (*Config, error) {
	fs := flag.NewFlagSet("fs", flag.ExitOnError)
	conf := Config{}
	fs.StringVar(&conf.ServerAddress, "a", "localhost:8080", "server address")
	fs.StringVar(&conf.ResponseURL, "b", "http://localhost:8080", "response URL")
	fs.StringVar(&conf.FileStoragePath, "f", "./data/local_storage.json", "file storage path")
	fs.StringVar(&conf.DebugLevel, "debug", "Info", "debug level")

	err := fs.Parse(os.Args[1:])
	if err != nil {
		return nil, err
	}

	envServerAddress, ok := os.LookupEnv("SERVER_ADDRESS")
	if ok {
		conf.ServerAddress = envServerAddress
	}

	envBaseURL, ok := os.LookupEnv("BASE_URL")
	if ok {
		conf.ResponseURL = envBaseURL
	}

	envDebugLevel, ok := os.LookupEnv("DEBUG_LEVEL")
	if ok {
		conf.DebugLevel = envDebugLevel
	}

	envFileStorePath, ok := os.LookupEnv("FILE_STORAGE_PATH")
	if ok {
		conf.FileStoragePath = envFileStorePath
	}

	return &conf, nil
}

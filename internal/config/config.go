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
	Mode            string `env:"MODE"`
}

func ParseFlags() (*Config, error) {
	fs := flag.NewFlagSet("fs", flag.ExitOnError)
	conf := Config{}
	fs.StringVar(&conf.ServerAddress, "a", "localhost:8080", "server address")
	fs.StringVar(&conf.ResponseURL, "b", "http://localhost:8080", "response URL")
	fs.StringVar(&conf.FileStoragePath, "f", "./data/local_storage.json", "file storage path")
	fs.StringVar(&conf.DebugLevel, "debug", "Info", "debug level")
	fs.StringVar(&conf.Mode, "mode", "dev", "mode")

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
	if envFileStorePath := os.Getenv("FILE_STORAGE_PATH"); envFileStorePath != "" {
		conf.FileStoragePath = envFileStorePath
	}
	return &conf, nil
}

func (conf *Config) UploadDB() string {
	return conf.FileStoragePath
}

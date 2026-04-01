package config

import (
	"database/sql"
	"flag"
	"os"
)

type Config struct {
	ServerAddress   string `env:"SERVER_ADDRESS"`
	ResponseURL     string `env:"BASE_URL"`
	DebugLevel      string `env:"DEBUG_LEVEL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	DatabaseDsn     string `env:"DATABASE_DSN"`
}

func ParseFlags() (*Config, error) {
	fs := flag.NewFlagSet("fs", flag.ExitOnError)
	conf := Config{}

	fs.StringVar(&conf.ServerAddress, "a", "localhost:8080", "server address")
	fs.StringVar(&conf.ResponseURL, "b", "http://localhost:8080", "response URL")
	fs.StringVar(&conf.FileStoragePath, "f", "", "file storage path")
	fs.StringVar(&conf.DebugLevel, "debug", "Info", "debug level")
	fs.StringVar(&conf.DatabaseDsn, "d", "", "database dsn")

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
	envDatabaseDsn, ok := os.LookupEnv("DATABASE_DSN")
	if ok {
		conf.DatabaseDsn = envDatabaseDsn
	}

	return &conf, nil
}

type App struct {
	DB  *sql.DB
	Cfg *Config
}

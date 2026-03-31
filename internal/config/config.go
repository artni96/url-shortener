package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/joho/godotenv"
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

	err := godotenv.Load()
	if err != nil {
		return nil, err
	}
	defaultDBDsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_USER_PASSWORD"), os.Getenv("DB_NAME"))

	fs.StringVar(&conf.ServerAddress, "a", "localhost:8080", "server address")
	fs.StringVar(&conf.ResponseURL, "b", "http://localhost:8080", "response URL")
	fs.StringVar(&conf.FileStoragePath, "f", "./data/local_storage.json", "file storage path")
	fs.StringVar(&conf.DebugLevel, "debug", "Info", "debug level")
	fs.StringVar(&conf.DatabaseDsn, "d", defaultDBDsn, "database dsn")

	err = fs.Parse(os.Args[1:])
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

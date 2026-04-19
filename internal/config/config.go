package config

import (
	"errors"
	"flag"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

type Config struct {
	ServerAddress   string        `env:"SERVER_ADDRESS"`
	ResponseDomain  string        `env:"BASE_URL"`
	DebugLevel      string        `env:"DEBUG_LEVEL"`
	FileStoragePath string        `env:"FILE_STORAGE_PATH"`
	DatabaseDsn     string        `env:"DATABASE_DSN"`
	SecretKey       string        `env:"SECRET_KEY"`
	TokenExp        time.Duration `env:"TOKEN_EXPIRATION"`
}

func ParseFlags() (*Config, error) {
	fs := flag.NewFlagSet("fs", flag.ExitOnError)
	conf := Config{}

	fs.StringVar(&conf.ServerAddress, "a", "localhost:8080", "server address")
	fs.StringVar(&conf.ResponseDomain, "b", "http://localhost:8080", "response URL")
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
		conf.ResponseDomain = envBaseURL
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
	err = godotenv.Load(".env")
	if err != nil {
		return &conf, errors.New("error loading .env file. Keep working with default values")
	}
	secretKey, ok := os.LookupEnv("SECRET_KEY")
	if ok {
		conf.SecretKey = secretKey
	} else {
		conf.SecretKey = "don't share me please"
	}
	tokenExp, ok := os.LookupEnv("TOKEN_EXPIRATION")
	if ok {
		conf.TokenExp, err = time.ParseDuration(tokenExp)
	} else {
		conf.TokenExp = time.Minute * 10
	}
	return &conf, nil
}

type App struct {
	DB     *sqlx.DB
	Cfg    *Config
	Logger *zap.Logger
}

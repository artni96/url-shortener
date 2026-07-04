package config

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/artni96/url-shortener/internal/model"
)

type jsonConfig struct {
	ServerAddress   string `json:"server_address"`
	BaseURL         string `json:"base_url"`
	FileStoragePath string `json:"file_storage_path"`
	DatabaseDSN     string `json:"database_dsn"`
	EnableHTTPS     bool   `json:"enable_https"`
}

type Config struct {
	ServerAddress   string        `env:"SERVER_ADDRESS"`
	ResponseDomain  string        `env:"BASE_URL"`
	DebugLevel      string        `env:"DEBUG_LEVEL"`
	FileStoragePath string        `env:"FILE_STORAGE_PATH"`
	DatabaseDsn     string        `env:"DATABASE_DSN"`
	SecretKey       string        `env:"SECRET_KEY"`
	TokenExp        time.Duration `env:"TOKEN_EXPIRATION"`
	AuditFile       string        `env:"AUDIT_FILE"`
	AuditURL        string        `env:"AUDIT_URL"`
	EnableHTTPS     bool          `env:"ENABLE_HTTPS"`
	CertFile        string        `env:"CERT_FILE"`
	KeyFile         string        `env:"KEY_FILE"`
	HostWhitelist   []string      `env:"HOST_WHITE_LIST"`
	Mode            string        `env:"MODE"`
}

func ParseFlags() (*Config, error) {
	fs := flag.NewFlagSet("fs", flag.ExitOnError)
	var jsonFilePath string
	var hostWhitelistStr string
	conf := Config{}

	fs.StringVar(&conf.ServerAddress, "a", "localhost:8080", "server address")
	fs.StringVar(&conf.ResponseDomain, "b", "http://localhost:8080", "response URL")
	fs.StringVar(&conf.FileStoragePath, "f", "", "file storage path")
	fs.StringVar(&conf.DebugLevel, "debug", "Info", "debug level")
	fs.StringVar(&conf.DatabaseDsn, "d", "", "database dsn")
	fs.StringVar(&conf.AuditFile, "audit-file", "", "audit file path")
	fs.StringVar(&conf.AuditURL, "audit-url", "", "audit url")
	fs.BoolVar(&conf.EnableHTTPS, "s", false, "enable https")
	fs.StringVar(&conf.CertFile, "cf", "", "cert file")
	fs.StringVar(&conf.KeyFile, "kf", "", "key file")
	fs.StringVar(&hostWhitelistStr, "hwl", "", "host whitelist")
	fs.StringVar(&conf.Mode, "m", "dev", "launching mode")
	fs.StringVar(&jsonFilePath, "c", "", "json config")

	err := fs.Parse(os.Args[1:])
	if err != nil {
		return nil, err
	}

	envJSONConfPath, ok := os.LookupEnv("JSON_CONFIG")
	if ok {
		jsonFilePath = envJSONConfPath
	}

	var jsonConf *jsonConfig

	if jsonFilePath != "" {
		jsonConf, err = readJSONConfig(jsonFilePath)
		if err != nil {
			return nil, err
		}

	}

	envServerAddress, ok := os.LookupEnv("SERVER_ADDRESS")
	if ok {
		conf.ServerAddress = envServerAddress
	} else if conf.ServerAddress == "localhost:8080" && jsonConf != nil && jsonConf.ServerAddress != "" && conf.ServerAddress != jsonConf.ServerAddress {
		conf.ServerAddress = jsonConf.ServerAddress
	}

	envBaseURL, ok := os.LookupEnv("BASE_URL")
	if ok {
		conf.ResponseDomain = envBaseURL
	} else if conf.ResponseDomain == "http://localhost:8080" && jsonConf != nil && jsonConf.BaseURL != "" && conf.ResponseDomain != jsonConf.BaseURL {
		conf.ResponseDomain = jsonConf.BaseURL
	}

	envDebugLevel, ok := os.LookupEnv("DEBUG_LEVEL")
	if ok {
		conf.DebugLevel = envDebugLevel
	}

	envFileStorePath, ok := os.LookupEnv("FILE_STORAGE_PATH")
	if ok {
		conf.FileStoragePath = envFileStorePath
	} else if conf.FileStoragePath == "" && jsonConf != nil && jsonConf.FileStoragePath != "" {
		conf.FileStoragePath = jsonConf.FileStoragePath
	}

	envDatabaseDsn, ok := os.LookupEnv("DATABASE_DSN")
	if ok {
		conf.DatabaseDsn = envDatabaseDsn
	} else if conf.DatabaseDsn == "" && jsonConf != nil && jsonConf.DatabaseDSN != "" {
		conf.DatabaseDsn = jsonConf.DatabaseDSN
	}

	envAuditFile, ok := os.LookupEnv("AUDIT_FILE")
	if ok {
		conf.AuditFile = envAuditFile
	}

	envAuditURL, ok := os.LookupEnv("AUDIT_URL")
	if ok {
		conf.AuditURL = envAuditURL
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
		if err != nil {
			return nil, err
		}
	} else {
		conf.TokenExp = time.Minute * 1
	}

	envEnableHTTPS, ok := os.LookupEnv("ENABLE_HTTPS")
	if ok {
		conf.EnableHTTPS = envEnableHTTPS == "true"
	} else if conf.EnableHTTPS == false && jsonConf != nil && jsonConf.EnableHTTPS {
		conf.EnableHTTPS = true
	}
	envCertFile, ok := os.LookupEnv("CERT_FILE")
	if ok {
		conf.CertFile = envCertFile
	} else {
		conf.CertFile = "./certs/local/127.0.0.1+1.pem"
	}
	envKeyFile, ok := os.LookupEnv("KEY_FILE")
	if ok {
		conf.KeyFile = envKeyFile
	} else {
		conf.KeyFile = "./certs/local/127.0.0.1+1-key.pem"
	}

	conf.HostWhitelist = strings.Split(hostWhitelistStr, ",")
	envHostWhitelist, ok := os.LookupEnv("HOST_WHITE_LIST")
	if ok {
		conf.HostWhitelist = strings.Split(envHostWhitelist, ",")
	}
	return &conf, nil
}

// generate:reset
type App struct {
	DB        *sqlx.DB
	Cfg       *Config
	Logger    *zap.Logger
	AuditChan chan model.AuditEntity
}

// readJSONConfig read config settings from a JSON file.
func readJSONConfig(path string) (*jsonConfig, error) {
	conf := &jsonConfig{}
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(file, &conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

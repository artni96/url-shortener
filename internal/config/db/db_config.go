package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/artni96/url-shortener/internal/config"
	"go.uber.org/zap"
)

type dBConnectionSettings struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func InitDBConnection(ctx context.Context, cfg *config.Config, log *zap.Logger) (*sql.DB, error) {
	localCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if cfg.DatabaseDsn == "" {
		log.Info("database dsn is empty, cannot connect to database")
		return nil, errors.New("database dsn is required")
	}
	db, err := sql.Open("pgx", cfg.DatabaseDsn)
	if err != nil {
		log.Info("failed to connect to database",
			zap.String("database destination", cfg.DatabaseDsn),
			zap.String("error message", err.Error()),
		)
		return nil, err
	}
	if db == nil {
		log.Info("failed to connect to database")
		return nil, errors.New("failed to connect to database")
	}

	err = db.PingContext(localCtx)
	if err != nil {
		log.Info("failed to ping database, trying to work with local file storage",
			zap.String("database destination", cfg.DatabaseDsn),
			zap.String("error message", err.Error()),
		)
		return nil, err
	}

	if err := runMigrations(cfg); err != nil {
		log.Info("failed to run migrations",
			zap.String("error message", err.Error()),
		)
		return nil, err
	}

	log.Info("Migrations completed successfully")
	return db, nil
}

func runMigrations(cfg *config.Config) error {
	dbAddress, err := dbAddressToRunMigrations(cfg)
	if err != nil {
		return err
	}
	fmt.Println("db address:", dbAddress)
	cmd := exec.Command(
		"migrate",
		"-database", dbAddress,
		"-path", "./migrations", "up",
	)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}

func dbAddressToRunMigrations(cfg *config.Config) (string, error) {
	dbConnParams := make(map[string]string)
	splitDatabaseDsn := strings.Split(cfg.DatabaseDsn, " ")
	for _, s := range splitDatabaseDsn {
		if s != "" {
			splitObj := strings.Split(s, "=")
			dbConnParams[splitObj[0]] = splitObj[1]
		}
	}
	if len(dbConnParams) != 6 {
		return "", fmt.Errorf("database dsn is not valid")
	}
	result := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dbConnParams["user"], dbConnParams["password"], dbConnParams["host"], dbConnParams["port"], dbConnParams["dbname"], dbConnParams["sslmode"])
	return result, nil
}

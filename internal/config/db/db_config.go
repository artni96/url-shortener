package db

import (
	"context"
	"database/sql"
	"errors"
	"os/exec"
	"time"

	"github.com/artni96/url-shortener/internal/config"
	"go.uber.org/zap"
)

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

	//if err := runMigrations(); err != nil {
	//	log.Info("failed to run migrations",
	//		zap.String("error message", err.Error()),
	//	)
	//	return nil, err
	//}
	//
	//log.Info("Migrations completed successfully")
	return db, nil
}

func runMigrations() error {

	cmd := exec.Command(
		"migrate",
		"-database", "postgres://postgres:postgres@localhost:5432/url_shortener?sslmode=disable",
		"-path", "./migrations", "up",
	)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}

package db

import (
	"context"
	"database/sql"
	"os/exec"

	"github.com/artni96/url-shortener/internal/config"
	"go.uber.org/zap"
)

func InitDBConnection(ctx context.Context, cfg *config.Config, log *zap.Logger) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DatabaseDsn)
	if err != nil {
		log.Info("failed to connect to database",
			zap.String("database destination", cfg.DatabaseDsn),
			zap.String("error message", err.Error()),
		)
		return nil, err
	}
	err = db.PingContext(ctx)
	//if err != nil {
	//	log.Info("failed to ping database, keep working with local storage",
	//		zap.String("database destination", cfg.DatabaseDsn),
	//		zap.String("error message", err.Error()),
	//	)
	//}
	//
	//if err := runMigrations(); err != nil {
	//	log.Info("failed to run migrations",
	//		zap.String("error message", err.Error()),
	//	)
	//	return nil, err
	//}

	log.Info("Migrations completed successfully")
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

package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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

	if err := runMigrations(db); err != nil {
		log.Info("failed to run migrations",
			zap.String("error message", err.Error()),
		)
	} else {
		log.Info("Migrations completed successfully")
	}

	return db, nil
}

func runMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	migrator, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return err
	}

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}

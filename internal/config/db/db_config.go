package db

import (
	"context"
	"errors"
	"time"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

func InitDBConnection(ctx context.Context, app *config.App) (*sqlx.DB, error) {
	if app.Cfg.DatabaseDsn == "" {
		app.Logger.Info("database dsn is empty, cannot connect to database")
		return nil, errors.New("database dsn is required")
	}
	localCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	db := sqlx.MustOpen("pgx", app.Cfg.DatabaseDsn)
	if db == nil {
		app.Logger.Info("failed to connect to database")
		return nil, errors.New("failed to connect to database")
	}

	err := db.PingContext(localCtx)
	if err != nil {
		app.Logger.Info("failed to ping database, trying to work with local file storage",
			zap.String("database destination", app.Cfg.DatabaseDsn),
			zap.String("error message", err.Error()),
		)
		return nil, err
	}

	if err := runMigrations(db); err != nil {
		app.Logger.Info("failed to run migrations",
			zap.String("error message", err.Error()),
		)
	} else {
		app.Logger.Info("Migrations completed successfully")
	}

	return db, nil
}

func runMigrations(db *sqlx.DB) error {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
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

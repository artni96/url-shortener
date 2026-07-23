package stats

import (
	"context"
	"fmt"

	"github.com/artni96/url-shortener/internal/model"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type DBStatsRepositoryInterface interface {
	Get(ctx context.Context) (model.StatsResponse, error)
}

type DBStatsRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func (repo *DBStatsRepository) Get(ctx context.Context) (model.StatsResponse, error) {
	tx, err := repo.db.BeginTxx(ctx, nil)
	if err != nil {
		return model.StatsResponse{}, fmt.Errorf("failed to start a transaction for getting stats: %w", err)
	}
	defer tx.Rollback()

	var urlsAmount int64
	urlsAmountQuery := "SELECT count(*) FROM urls"
	err = tx.GetContext(ctx, &urlsAmount, urlsAmountQuery)
	if err != nil {
		return model.StatsResponse{}, fmt.Errorf("failed to get urls amount: %w", err)
	}

	var usersAmount int64
	usersAmountQuery := "SELECT count(*) FROM users"
	err = tx.GetContext(ctx, &usersAmount, usersAmountQuery)
	if err != nil {
		return model.StatsResponse{}, fmt.Errorf("failed to get users amount: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return model.StatsResponse{}, fmt.Errorf("failed to finish transaction: %w", err)
	}

	return model.StatsResponse{URLs: urlsAmount, Users: usersAmount}, nil
}

func NewDBStatsRepository(db *sqlx.DB, logger *zap.Logger) (*DBStatsRepository, error) {
	return &DBStatsRepository{db: db, logger: logger}, nil
}

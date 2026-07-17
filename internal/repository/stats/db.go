package stats

import (
	"context"
	"fmt"

	"github.com/artni96/url-shortener/internal/config"
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
	var urlsAmount int64
	urlsAmountQuery := "SELECT count(*) FROM urls"
	err := repo.db.GetContext(ctx, &urlsAmount, urlsAmountQuery)
	if err != nil {
		return model.StatsResponse{}, fmt.Errorf("failed to get urls amount: %w", err)
	}

	var usersAmount int64
	usersAmountQuery := "SELECT count(*) FROM users"
	err = repo.db.GetContext(ctx, &usersAmount, usersAmountQuery)
	if err != nil {
		return model.StatsResponse{}, fmt.Errorf("failed to get users amount: %w", err)
	}
	return model.StatsResponse{URLs: urlsAmount, Users: usersAmount}, nil
}

func NewDBStatsRepository(app *config.App) (*DBStatsRepository, error) {
	return &DBStatsRepository{db: app.DB, logger: app.Logger}, nil
}

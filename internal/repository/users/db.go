package users

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
)

type DBUserRepositoryInterface interface {
	Create(ctx context.Context, ip string) (model.User, error)
	GetByIP(ctx context.Context, ip string) (int, error)
}

type DBUserRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func (repo *DBUserRepository) Create(ctx context.Context, ip string) (model.User, error) {
	var userID int

	insertQuery := "INSERT INTO users (ip) VALUES ($1) RETURNING id"

	var entity model.User
	err := repo.db.GetContext(ctx, &userID, insertQuery, ip)
	if err != nil {
		entity.IP = ip
		entity.ID = -1
		return entity, err
	}
	entity.ID = userID
	entity.IP = ip
	return entity, nil
}

func (repo *DBUserRepository) GetByIP(ctx context.Context, ip string) (int, error) {
	var userID int

	selectQuery := "SELECT id FROM users WHERE ip = $1"
	err := repo.db.GetContext(ctx, &userID, selectQuery, ip)
	if err != nil {
		return -1, fmt.Errorf("failed to get user by ip: %w", err)
	}
	return userID, nil
}

func NewUserDBRepository(app *config.App) (*DBUserRepository, error) {
	return &DBUserRepository{
		db:     app.DB,
		logger: app.Logger,
	}, nil
}

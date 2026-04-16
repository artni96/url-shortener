package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type DBAuthRepositoryInterface interface {
	Create(ctx context.Context, user model.UserCreate) (model.UserWithHashedPassword, error)
	GetUserHashedPassword(ctx context.Context, username string) (model.UserWithHashedPassword, error)
}

type DBAuthRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func (repo *DBAuthRepository) Create(ctx context.Context, user model.UserCreate) (model.UserWithHashedPassword, error) {
	responseEntity := model.UserWithHashedPassword{}

	insertQuery := "INSERT INTO users (username, password) VALUES ($1, $2) RETURNING id"

	result, err := repo.db.ExecContext(ctx, insertQuery, user.Username, user.HashedPassword)
	var uniqueConstrErr *pgconn.PgError
	if err != nil {
		if errors.As(err, &uniqueConstrErr) {
			return responseEntity, ErrUserAlreadyExists
		}
		return responseEntity, fmt.Errorf("failed to create new user: %w", err)
	}
	if result == nil {
		return responseEntity, ErrUserNotCreated
	}
	responseEntity.Username = user.Username
	responseEntity.Password = user.HashedPassword
	return responseEntity, nil
}

func (repo *DBAuthRepository) GetUserHashedPassword(ctx context.Context, username string) (model.UserWithHashedPassword, error) {
	selectQuery := "SELECT id, username, password FROM users WHERE username = $1"
	userResponse := model.UserWithHashedPassword{}
	err := repo.db.GetContext(ctx, &userResponse, selectQuery, username)
	if err != nil {
		return userResponse, fmt.Errorf("failed to get user hashed password: %w", err)
	}
	if userResponse.Username == "" {
		return userResponse, ErrUserNotFound
	}
	return userResponse, nil
}

func NewAuthDBRepository(app *config.App) (*DBAuthRepository, error) {
	return &DBAuthRepository{
		db:     app.DB,
		logger: app.Logger,
	}, nil
}

package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/artni96/url-shortener/internal/model"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type DBAuthRepositoryInterface interface {
	Create(ctx context.Context, user model.UserCreate) (string, error)
	GetUserHashedPassword(ctx context.Context, user model.UserLogin) (model.UserWithHashedPassword, error)
}

type DBAuthRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func (repo *DBAuthRepository) Create(ctx context.Context, user model.UserCreate) (string, error) {
	insertQuery := "INSERT INTO users (username, password) VALUES ($1, $2)"

	result, err := repo.db.ExecContext(ctx, insertQuery, user.Username, user.HashedPassword)
	var uniqueConstrErr *pgconn.PgError
	if err != nil {
		if errors.As(err, &uniqueConstrErr) {
			return "", ErrUserAlreadyExists
		}
		return "", fmt.Errorf("failed to create new user: %w", err)
	}
	if result == nil {
		return "", ErrUserNotCreated
	}
	return fmt.Sprintf("user %s successfully created", user.Username), nil
}

func (repo *DBAuthRepository) GetUserHashedPassword(ctx context.Context, user model.UserLogin) (model.UserWithHashedPassword, error) {
	selectQuery := "SELECT id, username, password FROM users WHERE username = $1"
	userResponse := model.UserWithHashedPassword{}
	err := repo.db.GetContext(ctx, &userResponse, selectQuery, user.Username)
	if err != nil {
		return userResponse, fmt.Errorf("failed to get user hashed password: %w", err)
	}
	if userResponse.Username == "" {
		return userResponse, ErrUserNotFound
	}
	return userResponse, nil

}

func NewAuthDBRepository(db *sqlx.DB, logger *zap.Logger) *DBAuthRepository {
	return &DBAuthRepository{
		db:     db,
		logger: logger,
	}
}

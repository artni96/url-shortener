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
	GetUserHashedPassword(ctx context.Context, user model.UserLogin) (string, error)
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
		return "", err
	}
	if result == nil {
		return "", ErrUserNotCreated
	}
	return fmt.Sprintf("user %s successfully created", user.Username), nil
}

func (repo *DBAuthRepository) GetUserHashedPassword(ctx context.Context, user model.UserLogin) (string, error) {
	selectQuery := "SELECT password FROM users WHERE username = $1"
	hashedPassword := ""
	err := repo.db.GetContext(ctx, &hashedPassword, selectQuery, user.Username)
	if err != nil {
		return "", fmt.Errorf("failed to get user hashed password: %w", err)
	}
	if hashedPassword == "" {
		return "", ErrUserNotFound
	}
	return hashedPassword, nil

}

func NewAuthDBRepository(db *sqlx.DB, logger *zap.Logger) *DBAuthRepository {
	return &DBAuthRepository{
		db:     db,
		logger: logger,
	}
}

package users

import (
	"context"
	"fmt"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type DBUserRepositoryInterface interface {
	Create(ctx context.Context) (int, error)
	//GetUserHashedPassword(ctx context.Context, username string) (model.UserWithHashedPassword, error)
}

type DBUserRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func (repo *DBUserRepository) Create(ctx context.Context) (int, error) {
	var userID int

	insertQuery := "INSERT INTO users DEFAULT VALUES RETURNING id"

	err := repo.db.GetContext(ctx, &userID, insertQuery)
	if err != nil {
		return -1, err
	}
	//var uniqueConstrErr *pgconn.PgError
	//if err != nil {
	//	if errors.As(err, &uniqueConstrErr) {
	//		return responseEntity, ErrUserAlreadyExists
	//	}
	//	return responseEntity, fmt.Errorf("failed to create new user: %w", err)
	//}
	//if result == nil {
	//	return responseEntity, ErrUserNotCreated
	//}
	//responseEntity.Username = user.Username
	//responseEntity.Password = user.HashedPassword
	return userID, nil
}

func (repo *DBUserRepository) GetUserHashedPassword(ctx context.Context, username string) (model.UserWithHashedPassword, error) {
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

func NewUserDBRepository(app *config.App) (*DBUserRepository, error) {
	return &DBUserRepository{
		db:     app.DB,
		logger: app.Logger,
	}, nil
}

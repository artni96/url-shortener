package users

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/artni96/url-shortener/internal/model"
)

// DBUserRepositoryInterface encapsulates the logic to handle User entities via database management system (Postgres).
type DBUserRepositoryInterface interface {
	// Create saves a User entity.
	Create(ctx context.Context, ip string) (model.User, error)
	// GetByIP returns a User entity by its ID.
	GetByIP(ctx context.Context, ip string) (int, error)
}

// DBUserRepository implements an object-mediator with a database.
type DBUserRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// Create saves a new User entity in the database by user's IP.
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

// GetByIP returns a User data from the database by its ID.
func (repo *DBUserRepository) GetByIP(ctx context.Context, ip string) (int, error) {
	var userID int

	selectQuery := "SELECT id FROM users WHERE ip = $1"
	err := repo.db.GetContext(ctx, &userID, selectQuery, ip)
	if err != nil {
		return -1, fmt.Errorf("failed to get user by ip: %w", err)
	}
	return userID, nil
}

// NewUserDBRepository initializes a new DBUserRepository.
func NewUserDBRepository(db *sqlx.DB, logger *zap.Logger) (*DBUserRepository, error) {
	return &DBUserRepository{
		db:     db,
		logger: logger,
	}, nil
}

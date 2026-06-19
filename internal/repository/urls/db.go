package urls

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
)

// DBURLRepositoryInterface encapsulates the logic to handle URL entities via database management system (Postgres).
type DBURLRepositoryInterface interface {
	// Create saves a new URL entity.
	Create(ctx context.Context, url model.URLCreate) (model.URLEntity, error)
	// BulkCreate saves several URL entities by one commit.
	BulkCreate(ctx context.Context, urls []model.URLBulkCreate) ([]model.URLBulkCreate, error)
	// GetByShortURL returns a URL entity with original URL value by its short URL.
	GetByShortURL(ctx context.Context, shortURL string) (model.GetByShortURLResponse, error)
	// GetList returns the list of all URL entities.
	GetList(ctx context.Context) ([]model.URLEntity, error)
	// GetUserList returns the list of user's URL entities.
	GetUserList(ctx context.Context, createdBy int) ([]model.URLEntity, error)
	// Update saves the changes of a URL entity gotten by its short URL.
	Update(ctx context.Context, url model.URLEntity) (model.URLEntity, error)
	// Delete removes a URL entity by its short URL value.
	Delete(ctx context.Context, shortURL string) error
	// BulkDelete removes several URL entities by one commit.
	BulkDelete(ctx context.Context, urls []model.URLDelete) ([]error, error)

	// IsURLListUnique checks whether the whole list of generated short URL values is unique.
	IsURLListUnique(ctx context.Context, shortURLList []string) (bool, error)
}

// DBURLRepository is an object to manipulate URL data via database management system (Postgres).
type DBURLRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// Create saves a new URL entity. It returns a URLEntity object.
func (repo *DBURLRepository) Create(ctx context.Context, requestEntity model.URLCreate) (model.URLEntity, error) {

	responseEntity := model.URLEntity{}
	selectQuery := "SELECT original_url, short_url FROM urls WHERE short_url = $1"
	err := repo.db.GetContext(ctx, &responseEntity, selectQuery, requestEntity.ShortURL)

	if err.Error() != "sql: no rows in result set" {
		return model.URLEntity{}, err
	}

	if responseEntity.ShortURL != "" && responseEntity.ShortURL != requestEntity.ShortURL {
		return model.URLEntity{}, fmt.Errorf("%w: short url already exists", ErrShortURLAlreadyExists)
	}
	var uniqueConstrErr *pgconn.PgError
	insertQuery := "INSERT INTO urls (original_url, short_url, created_by, is_deleted) VALUES ($1, $2, $3, $4)"

	result, err := repo.db.ExecContext(ctx, insertQuery, requestEntity.OriginalURL, requestEntity.ShortURL, requestEntity.CreatedBy, false)
	if err != nil {
		if errors.As(err, &uniqueConstrErr) {
			selectQuery = "SELECT original_url, short_url FROM urls WHERE original_url = $1"
			err = repo.db.GetContext(ctx, &responseEntity, selectQuery, requestEntity.OriginalURL)
			if err != nil {
				return model.URLEntity{}, fmt.Errorf("failed to insert url: %w", err)
			}
			return responseEntity, fmt.Errorf("%w: %s", ErrOriginalURLAlreadyExists, requestEntity.OriginalURL)
		}
	}

	if result == nil {
		return model.URLEntity{}, ErrURLNotCreated
	}
	responseEntity.OriginalURL = requestEntity.OriginalURL
	responseEntity.ShortURL = requestEntity.ShortURL
	responseEntity.CreatedBy = requestEntity.CreatedBy
	return responseEntity, nil
}

// BulkCreate saves several URL entities by one commit. It returns created URLBulkCreate entities.
func (repo *DBURLRepository) BulkCreate(ctx context.Context, urls []model.URLBulkCreate) ([]model.URLBulkCreate, error) {

	tx, err := repo.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("BulkCreate - failure to begin transaction: %w", err)
	}

	defer tx.Rollback()

	query := "INSERT INTO urls (original_url, short_url, created_by, is_deleted) VALUES "
	var values []interface{}
	var result []model.URLBulkCreate

	for i, url := range urls {
		if i > 0 {
			query += ", "
		}
		query += fmt.Sprintf("($%d, $%d, $%d, $%d)", len(values)+1, len(values)+2, len(values)+3, len(values)+4)
		values = append(values, url.OriginalURL, url.ShortURL, url.CreatedBy, false)
		result = append(result, model.URLBulkCreate{
			CorrelationID: url.CorrelationID,
			ShortURL:      url.ShortURL,
		})
	}
	var uniqueConstrErr *pgconn.PgError
	_, err = tx.ExecContext(ctx, query, values...)
	if err != nil {
		if errors.As(err, &uniqueConstrErr) {
			return nil, ErrDuplicatedURL
		}
		return nil, fmt.Errorf("failed to bulk create: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit bulk create: %w", err)
	}
	return result, nil
}

// BulkDelete removes several URL entities by one commit.
// It returns a slice of errors - error at removing an exact URL entity.
func (repo *DBURLRepository) BulkDelete(ctx context.Context, urls []model.URLDelete) ([]error, error) {
	doneChan := make(chan struct{})
	defer close(doneChan)

	inputChan := generator(doneChan, urls)
	channels := fanOut(ctx, doneChan, inputChan, repo, len(urls))

	resultChan := fanIn(doneChan, channels...)

	var urlList []string
	var errs []error

	for url := range resultChan {
		if url.Err != nil {
			errs = append(errs, url.Err)
		} else {
			urlList = append(urlList, url.ShortURL)
		}
	}

	query := "UPDATE urls SET is_deleted = TRUE WHERE short_url = ANY($1)"
	_, err := repo.db.ExecContext(ctx, query, urlList)
	if err != nil {
		return errs, err
	}
	return errs, nil
}

// GetByShortURL returns a URL entity with original URL value by its short URL.
func (repo *DBURLRepository) GetByShortURL(ctx context.Context, shortURL string) (model.GetByShortURLResponse, error) {
	var entity model.GetByShortURLResponse

	selectQuery := "SELECT original_url, short_url, is_deleted FROM urls where short_url = $1"
	err := repo.db.GetContext(ctx, &entity, selectQuery, shortURL)

	if err != nil {
		return model.GetByShortURLResponse{}, fmt.Errorf("%w: %s", ErrURLNotFound, shortURL)
	}
	return entity, nil
}

// GetList returns the list of all URL entities.
func (repo *DBURLRepository) GetList(ctx context.Context) ([]model.URLEntity, error) {
	var entities []model.URLEntity

	selectQuery := "SELECT original_url, short_url FROM urls"
	err := repo.db.SelectContext(ctx, &entities, selectQuery)

	if err != nil {
		return nil, fmt.Errorf("GetList - failure to execute request: %w", err)
	}
	return entities, nil
}

// GetUserList returns the list of user's URL entities.
func (repo *DBURLRepository) GetUserList(ctx context.Context, createdBy int) ([]model.URLEntity, error) {
	var entities []model.URLEntity

	selectQuery := "SELECT original_url, short_url FROM urls where created_by = $1"
	err := repo.db.SelectContext(ctx, &entities, selectQuery, createdBy)

	if err != nil {
		return nil, fmt.Errorf("GetList - failure to execute request: %w", err)
	}
	return entities, nil
}

// Update saves the changes of a URL entity gotten by its short URL.
func (repo *DBURLRepository) Update(ctx context.Context, entity model.URLEntity) (model.URLEntity, error) {
	updatedEntity := model.URLEntity{}

	query := "UPDATE urls SET original_url = $1 WHERE short_url = $2 RETURNING original_url"
	stmt, err := repo.db.PrepareContext(ctx, query)

	if err != nil {
		return updatedEntity, fmt.Errorf("%w", err)
	}

	var uniqueConstrErr *pgconn.PgError
	resp, err := stmt.ExecContext(ctx, entity.ShortURL, entity.OriginalURL)
	if err != nil {
		if errors.As(err, &uniqueConstrErr) {
			selectQuery := "SELECT original_url, short_url FROM urls WHERE original_url = $1"
			repo.db.GetContext(ctx, &entity, selectQuery, entity.OriginalURL)
			return entity, fmt.Errorf("%w: original url: %s, short url: %s", ErrOriginalURLAlreadyExists, entity.OriginalURL, entity.ShortURL)
		}
		return updatedEntity, fmt.Errorf("%w", fmt.Errorf("failed to update url entity: %w", err))
	}

	_, err = resp.RowsAffected()
	if err != nil {
		return updatedEntity, fmt.Errorf("%w", ErrURLNotFound)
	}
	updatedEntity.OriginalURL = entity.OriginalURL
	updatedEntity.ShortURL = entity.ShortURL
	return updatedEntity, nil
}

// Delete removes a URL entity by its short URL value.
func (repo *DBURLRepository) Delete(ctx context.Context, shortURL string) error {
	query := "DELETE FROM urls WHERE short_url = $1"
	stmt, err := repo.db.PrepareContext(ctx, query)

	if err != nil {
		return fmt.Errorf("delete - failed to prepare statement: %w", err)
	}
	resp, err := stmt.ExecContext(ctx, shortURL)

	if err != nil {
		return fmt.Errorf("delete - failed to execute request: %w", err)
	}

	isRemoved, err := resp.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w", fmt.Errorf("delete - failed to get rows affected: %w", err))
	}
	if isRemoved == 0 {
		return fmt.Errorf("%w", ErrURLNotFound)
	}

	return nil
}

// IsURLListUnique checks whether the whole list of generated short URL values is unique.
func (repo *DBURLRepository) IsURLListUnique(ctx context.Context, shortURLList []string) (bool, error) {
	placeholders := make([]string, len(shortURLList))
	args := make([]interface{}, len(shortURLList))
	for i, url := range shortURLList {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = url
	}

	query := fmt.Sprintf(
		"SELECT short_url FROM urls WHERE short_url IN (%s)",
		strings.Join(placeholders, ", "),
	)

	rows, err := repo.db.QueryContext(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("failed to query URLs: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		return false, ErrURLListIsNotUnique
	}

	return true, rows.Err()
}

// NewDBURLRepository returns a new DBURLRepository.
func NewDBURLRepository(app *config.App) (*DBURLRepository, error) {
	return &DBURLRepository{
		db:     app.DB,
		logger: app.Logger,
	}, nil
}

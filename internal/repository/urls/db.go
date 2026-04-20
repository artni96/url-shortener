package urls

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type DBURLRepositoryInterface interface {
	Create(ctx context.Context, url model.URLCreate) (model.URLEntity, error)
	BulkCreate(ctx context.Context, urls []model.URLBulkCreate) ([]model.URLBulkCreate, error)
	GetByShortURL(ctx context.Context, shortURL string) (model.GetByShortURLResponse, error)
	GetList(ctx context.Context) ([]model.URLEntity, error)
	GetUserList(ctx context.Context, createdBy int) ([]model.URLEntity, error)
	Update(ctx context.Context, url model.URLEntity) (model.URLEntity, error)
	Delete(ctx context.Context, shortURL string) error
	BulkDelete(ctx context.Context, urls []model.URLDelete) ([]error, error)

	IsURLListUnique(ctx context.Context, shortURLList []string) (bool, error)
}
type DBURLRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

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

func (repo *DBURLRepository) BulkDelete(ctx context.Context, urls []model.URLDelete) ([]error, error) {
	doneChan := make(chan struct{})
	defer close(doneChan)

	inputChan := generator(doneChan, urls)
	channels := fanOut(ctx, doneChan, inputChan, repo)

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

func generator(doneCh chan struct{}, urls []model.URLDelete) chan model.URLDelete {
	inCh := make(chan model.URLDelete)

	go func() {
		defer close(inCh)

		for _, url := range urls {
			select {
			case <-doneCh:
				return
			case inCh <- url:

			}
		}
	}()
	return inCh
}

type Result struct {
	ShortURL string
	Err      error
}

func canBeDeleted(ctx context.Context, doneCh chan struct{}, inCh chan model.URLDelete, repo *DBURLRepository) chan Result {
	res := make(chan Result)

	go func() {
		defer close(res)

		for url := range inCh {

			urlData := Result{
				ShortURL: url.ShortURL,
				Err:      nil,
			}
			var createdBy int
			selectQuery := "SELECT created_by FROM urls WHERE short_url = $1"
			err := repo.db.GetContext(ctx, &createdBy, selectQuery, url.ShortURL)

			if err != nil {
				if errors.As(err, &pgx.ErrNoRows) {
					urlData.Err = fmt.Errorf("%w: %s", ErrURLNotFound, url.ShortURL)
				}
			}

			if urlData.Err == nil && createdBy != url.CreatedBy {
				urlData.Err = fmt.Errorf("%w: url author id: %d, request user id: %d", ErrUserIsNotAuthor, createdBy, url.CreatedBy)
			}

			select {
			case <-doneCh:
				return
			case res <- urlData:

			}
		}
	}()
	return res
}

func fanOut(ctx context.Context, doneCh chan struct{}, inCh chan model.URLDelete, repo *DBURLRepository) []chan Result {
	numWorkers := 5

	channels := make([]chan Result, numWorkers)

	for i := 0; i < numWorkers; i++ {
		canBeDeletedCh := canBeDeleted(ctx, doneCh, inCh, repo)
		channels[i] = canBeDeletedCh
	}
	return channels
}

func fanIn(doneCh chan struct{}, resultChs ...chan Result) chan Result {
	finalCh := make(chan Result)

	var wg sync.WaitGroup
	for _, ch := range resultChs {
		wg.Add(1)

		chClosure := ch

		go func() {
			defer wg.Done()

			for data := range chClosure {
				select {
				case <-doneCh:
					return
				case finalCh <- data:
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(finalCh)
	}()
	return finalCh
}

func (repo *DBURLRepository) GetByShortURL(ctx context.Context, shortURL string) (model.GetByShortURLResponse, error) {
	var entity model.GetByShortURLResponse

	selectQuery := "SELECT original_url, short_url, is_deleted FROM urls where short_url = $1"
	err := repo.db.GetContext(ctx, &entity, selectQuery, shortURL)

	if err != nil {
		return model.GetByShortURLResponse{}, fmt.Errorf("%w: %s", ErrURLNotFound, shortURL)
	}
	return entity, nil
}

func (repo *DBURLRepository) GetList(ctx context.Context) ([]model.URLEntity, error) {
	var entities []model.URLEntity

	selectQuery := "SELECT original_url, short_url FROM urls"
	err := repo.db.SelectContext(ctx, &entities, selectQuery)

	if err != nil {
		return nil, fmt.Errorf("GetList - failure to execute request: %w", err)
	}
	return entities, nil
}

func (repo *DBURLRepository) GetUserList(ctx context.Context, createdBy int) ([]model.URLEntity, error) {
	var entities []model.URLEntity

	selectQuery := "SELECT original_url, short_url FROM urls where created_by = $1"
	err := repo.db.SelectContext(ctx, &entities, selectQuery, createdBy)

	if err != nil {
		return nil, fmt.Errorf("GetList - failure to execute request: %w", err)
	}
	return entities, nil
}

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

func (repo *DBURLRepository) IsURLListUnique(ctx context.Context, shortURLList []string) (bool, error) {
	query := "SELECT original_url FROM urls WHERE short_url IN ($1)"
	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return false, fmt.Errorf("failed to prepare statement: %w", err)
	}
	result, err := stmt.QueryContext(ctx, shortURLList)
	if err != nil {
		return false, fmt.Errorf("failed to execute request: %w", err)
	}
	if result != nil {
		return false, ErrURLListIsNotUnique
	}
	return true, nil
}

func NewDBURLRepository(app *config.App) (*DBURLRepository, error) {
	return &DBURLRepository{
		db:     app.DB,
		logger: app.Logger,
	}, nil
}

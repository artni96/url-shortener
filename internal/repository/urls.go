package repository

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

var ErrShortURLAlreadyExists = errors.New("short url already exists")
var ErrOriginalURLAlreadyExists = errors.New("url already exists")
var ErrURLNotFound = errors.New("url not found")
var ErrURLNotCreated = errors.New("could not create url")
var ErrURLListIsNotUnique = errors.New("url list is not unique")
var ErrDuplicatedURL = errors.New(fmt.Sprintf("duplicated url"))

type URLRepositoryInterface interface {
	GetList(ctx context.Context) ([]model.URLEntity, error)
	GetByShortURL(ctx context.Context, shortURL string) (string, error)
	Create(ctx context.Context, originalURL, shortURL string) (model.URLEntity, error)
	BulkCreate(ctx context.Context, urls []model.URLBulkCreate) ([]model.URLBulkCreate, error)
	Update(ctx context.Context, entity model.URLEntity) (model.URLEntity, error)
	Delete(ctx context.Context, shortURL string) error

	SaveURLToLocalStorage(url model.URLEntity) error
	IsURLListUnique(ctx context.Context, shortURLList []string) (bool, error)
}

type LocalURLRepository struct {
	mu     sync.RWMutex
	urls   map[string]string
	db     *sqlx.DB
	logger *zap.Logger
}

func (repo *LocalURLRepository) Create(ctx context.Context, originalURL, shortURL string) (model.URLEntity, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.db != nil {
		entity := model.URLEntity{}
		selectQuery := "SELECT original_url, short_url FROM urls WHERE short_url = $1"
		err := repo.db.GetContext(ctx, &entity, selectQuery, shortURL)

		if err.Error() != "sql: no rows in result set" {
			return model.URLEntity{}, err
		}

		if entity.ShortURL != "" && entity.ShortURL != shortURL {
			return model.URLEntity{}, fmt.Errorf("%w: short url already exists", ErrShortURLAlreadyExists)
		}
		var uniqueConstrErr *pgconn.PgError
		insertQuery := "INSERT INTO urls (original_url, short_url) VALUES ($1, $2)"

		result, err := repo.db.ExecContext(ctx, insertQuery, originalURL, shortURL)
		if err != nil {
			if errors.As(err, &uniqueConstrErr) {

				selectQuery = "SELECT original_url, short_url FROM urls WHERE original_url = $1"
				repo.db.GetContext(ctx, &entity, selectQuery, originalURL)

				return entity, fmt.Errorf("%w: %s", ErrOriginalURLAlreadyExists, originalURL)
			}
		}
		if result == nil {
			return model.URLEntity{}, ErrURLNotCreated
		}

		entity.OriginalURL = originalURL
		entity.ShortURL = shortURL
		return entity, nil
	}

	if _, ok := repo.urls[originalURL]; ok {
		return model.URLEntity{}, ErrShortURLAlreadyExists
	}

	for _, url := range repo.urls {
		if url == originalURL {
			return model.URLEntity{}, ErrOriginalURLAlreadyExists
		}
	}

	repo.urls[shortURL] = originalURL
	entity := model.URLEntity{
		OriginalURL: originalURL,
		ShortURL:    shortURL,
	}
	return entity, nil

}

func (repo *LocalURLRepository) GetByShortURL(ctx context.Context, shortURL string) (string, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	var entity string

	if repo.db != nil {
		selectQuery := "SELECT original_url FROM urls where short_url = $1"
		err := repo.db.GetContext(ctx, &entity, selectQuery, shortURL)

		if err != nil {
			return "", fmt.Errorf("%w: %s", ErrURLNotFound, shortURL)
		}
		return entity, nil
	}

	entity, ok := repo.urls[shortURL]
	if !ok {
		return "", ErrURLNotFound
	}

	return entity, nil
}

func (repo *LocalURLRepository) GetList(ctx context.Context) ([]model.URLEntity, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	var entities []model.URLEntity

	if repo.db != nil {
		selectQuery := "SELECT original_url, short_url FROM urls"
		err := repo.db.SelectContext(ctx, &entities, selectQuery)

		if err != nil {
			return nil, fmt.Errorf("GetList - failure to execute request: %w", err)
		}
		return entities, nil
	}

	for shortURL, originalURL := range repo.urls {
		entities = append(entities, model.URLEntity{OriginalURL: originalURL, ShortURL: shortURL})
	}
	return entities, nil
}

func (repo *LocalURLRepository) BulkCreate(ctx context.Context, urls []model.URLBulkCreate) ([]model.URLBulkCreate, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	//var errs []error

	var result []model.URLBulkCreate

	urlDuplicates := map[string]int{}
	var duplicateErrs []error

	if repo.db != nil {
		tx, err := repo.db.BeginTxx(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("BulkCreate - failure to begin transaction: %w", err)
		}

		defer tx.Rollback()

		query := "INSERT INTO urls (original_url, short_url) VALUES "
		values := []interface{}{}
		result := []model.URLBulkCreate{}

		for i, url := range urls {
			if i > 0 {
				query += ", "
			}
			query += fmt.Sprintf("($%d, $%d)", len(values)+1, len(values)+2)
			values = append(values, url.OriginalURL, url.ShortURL)
			result = append(result, model.URLBulkCreate{
				OriginalURL: url.OriginalURL,
				ShortURL:    url.ShortURL,
			})
		}
		var uniqueConstrErr *pgconn.PgError
		_, err = tx.ExecContext(ctx, query, values...)
		if err != nil {
			if errors.As(err, &uniqueConstrErr) {
				return nil, fmt.Errorf("duplicate urls: %w", err)
			}
			return nil, fmt.Errorf("failed to bulk create: %w", err)
		}
		return result, tx.Commit()
	}

	//	tx, err := repo.db.BeginTxx(ctx, nil)
	//	if err != nil {
	//		return nil, fmt.Errorf("BulkCreate - failed to begin transaction: %w", err)
	//	}
	//
	//	query := "INSERT INTO urls (short_url, original_url) VALUES ($1, $2)"
	//	stmt, err := tx.PrepareContext(ctx, query)
	//	if err != nil {
	//		return nil, fmt.Errorf("BulkCreate - failed to prepare statement: %w", err)
	//	}
	//	var uniqueConstrErr *pgconn.PgError
	//	for _, url := range urls {
	//		_, err = stmt.ExecContext(ctx, url.ShortURL, url.OriginalURL)
	//		if err != nil {
	//			if errors.As(err, &uniqueConstrErr) {
	//				errs = append(errs, fmt.Errorf("%w: %s", ErrOriginalURLAlreadyExists, url.OriginalURL))
	//			}
	//			continue
	//
	//		}
	//		entity := model.URLBulkCreate{
	//			CorrelationID: url.CorrelationID,
	//			ShortURL:      url.ShortURL,
	//			OriginalURL:   url.OriginalURL,
	//		}
	//		result = append(result, entity)
	//		urlDuplicates[entity.OriginalURL] += 1
	//	}
	//	if len(errs) > 0 {
	//		err := tx.Rollback()
	//		if err != nil {
	//			return nil, fmt.Errorf("%w", err)
	//		}
	//		return nil, errors.Join(errs...)
	//	}
	//
	//	for url, count := range urlDuplicates {
	//		if count > 1 {
	//			duplicateErrs = append(duplicateErrs, fmt.Errorf("%w: %s", ErrDuplicatedURL, url))
	//		}
	//	}
	//	if len(duplicateErrs) > 0 {
	//		err := tx.Rollback()
	//		if err != nil {
	//			return nil, fmt.Errorf("%w", err)
	//		}
	//		return nil, errors.Join(duplicateErrs...)
	//	}
	//	return result, tx.Commit()
	//}

	for _, url := range urls {
		urlDuplicates[url.OriginalURL] += 1
	}

	for url, count := range urlDuplicates {
		if count > 1 {
			duplicateErrs = append(duplicateErrs, fmt.Errorf("%w: %s", ErrDuplicatedURL, url))
		}
	}
	if len(duplicateErrs) > 0 {
		return nil, errors.Join(duplicateErrs...)
	}

	for _, url := range urls {
		repo.urls[url.ShortURL] = url.OriginalURL
		entity := model.URLBulkCreate{
			CorrelationID: url.CorrelationID,
			ShortURL:      url.ShortURL,
			OriginalURL:   url.OriginalURL,
		}
		result = append(result, entity)
	}

	return result, nil
}

func (repo *LocalURLRepository) Update(ctx context.Context, entity model.URLEntity) (model.URLEntity, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	updatedEntity := model.URLEntity{}
	if repo.db != nil {
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
				return entity, fmt.Errorf("%w: %s", ErrOriginalURLAlreadyExists, model.URLEntity{
					OriginalURL: entity.OriginalURL,
					ShortURL:    entity.ShortURL,
				},
				)
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

	for _, originalURL := range repo.urls {
		if originalURL == entity.OriginalURL {
			return updatedEntity, ErrOriginalURLAlreadyExists
		}
	}

	for dbShortURL := range repo.urls {
		if dbShortURL == entity.ShortURL {
			repo.urls[dbShortURL] = entity.OriginalURL
			updatedEntity.OriginalURL = entity.OriginalURL
			updatedEntity.ShortURL = entity.ShortURL
			return updatedEntity, nil
		}
	}
	return updatedEntity, ErrURLNotFound
}

func (repo *LocalURLRepository) Delete(ctx context.Context, shortURL string) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.db != nil {

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

	for url := range repo.urls {
		if url == shortURL {
			delete(repo.urls, shortURL)
			return nil
		}
	}
	return fmt.Errorf("%w", ErrURLNotFound)
}

func (repo *LocalURLRepository) IsURLListUnique(ctx context.Context, shortURLList []string) (bool, error) {
	if repo.db != nil {
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
	for _, shortURL := range shortURLList {
		if _, ok := repo.urls[shortURL]; ok {
			return false, ErrURLListIsNotUnique
		}
	}
	return true, nil
}

func (repo *LocalURLRepository) SaveURLToLocalStorage(urlEntity model.URLEntity) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	_, ok := repo.urls[urlEntity.ShortURL]
	if ok {
		return fmt.Errorf("%w: %s", ErrShortURLAlreadyExists, urlEntity.ShortURL)
	}
	repo.urls[urlEntity.ShortURL] = urlEntity.OriginalURL
	return nil
}

type Writer struct {
	file   *os.File
	writer *bufio.Writer
}

func NewWriter(filename string) (*Writer, error) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("could open file: %w", err)
	}
	return &Writer{file: file, writer: bufio.NewWriter(file)}, nil
}

func (w *Writer) WriteEntity(entity *model.URLEntity) error {
	data, err := json.Marshal(&entity)
	if err != nil {
		return fmt.Errorf("could not marshal entity: %w", err)
	}

	if _, err = w.writer.Write(data); err != nil {
		return fmt.Errorf("could not write to file: %w", err)
	}

	if err = w.writer.WriteByte('\n'); err != nil {
		return fmt.Errorf("could not write to file: %w", err)
	}
	defer w.Close()
	return w.writer.Flush()
}

func (w *Writer) BulkWriteEntities(entities []model.URLEntity, toTruncate bool) error {
	if toTruncate {
		err := os.Truncate(w.file.Name(), 0)
		if err != nil {
			return fmt.Errorf("could not truncate file at ulr removal: %w", err)
		}
	}

	defer w.Close()
	for _, entity := range entities {
		entityForFile := model.URLEntity{
			OriginalURL: entity.OriginalURL,
			ShortURL:    entity.ShortURL,
		}
		data, err := json.Marshal(&entityForFile)
		if err != nil {
			return fmt.Errorf("could not marshal entity: %w", err)
		}

		if _, err = w.writer.Write(data); err != nil {
			return fmt.Errorf("could not write to file: %w", err)
		}

		if err = w.writer.WriteByte('\n'); err != nil {
			return fmt.Errorf("could not write to file: %w", err)
		}
		defer w.Close()
	}
	return w.writer.Flush()
}

func (w *Writer) Close() error {
	return w.file.Close()
}

type FileScanner struct {
	file    *os.File
	scanner *bufio.Scanner
}

func NewFileScanner(filename string) (*FileScanner, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	return &FileScanner{file: file, scanner: bufio.NewScanner(file)}, nil
}

func (s *FileScanner) Close() error {
	return s.file.Close()
}

func (s *FileScanner) CollectData() ([]model.URLEntity, error) {
	var result []model.URLEntity
	for s.scanner.Scan() {

		data := s.scanner.Bytes()

		object := model.URLEntity{}
		if err := json.Unmarshal(data, &object); err != nil {
			return nil, fmt.Errorf("could not unmarshal object: %w", err)
		}
		result = append(result, object)
	}
	return result, nil
}

func (s *FileScanner) CollectFilteredData(shortURL string) ([]model.URLBulkCreate, error) {
	var result []model.URLBulkCreate
	for s.scanner.Scan() {

		data := s.scanner.Bytes()

		object := model.URLBulkCreate{}
		if err := json.Unmarshal(data, &object); err != nil {
			return nil, fmt.Errorf("could not unmarshal object: %w", err)
		}
		if object.ShortURL != shortURL {
			result = append(result, object)
		}
	}
	return result, nil
}

func (repo *LocalURLRepository) uploadLocalStorage(filepath string) error {
	fileReader, err := NewFileScanner(filepath)
	if fileReader == nil {
		return nil
	}
	defer func(fileReader *FileScanner) {
		err := fileReader.Close()
		if err != nil {
			repo.logger.Info("could not close file reader", zap.String("filepath", filepath))
		}
	}(fileReader)

	if err != nil {
		repo.logger.Info("could not initialize NewFileScanner",
			zap.String("error", err.Error()))
		return nil
	}
	result, err := fileReader.CollectData()
	if err != nil {
		repo.logger.Info("could not collect data from file",
			zap.String("filepath", filepath),
			zap.String("error", err.Error()))
		return fmt.Errorf("could not collect data from file: %w", err)
	}
	for _, object := range result {
		err := repo.SaveURLToLocalStorage(object)
		if err != nil {
			repo.logger.Info("could not upload URL Entity to local storage",
				zap.String("ShortURL", object.ShortURL),
				zap.String("OriginalURL", object.OriginalURL))
			return fmt.Errorf("could not upload URL Entity to local storage: %w", err)
		}
	}
	return nil
}

func NewURLRepository(app *config.App) (*LocalURLRepository, error) {
	repo := LocalURLRepository{
		urls:   make(map[string]string),
		db:     app.DB,
		logger: app.Logger,
	}

	if repo.db == nil && app.Cfg.FileStoragePath != "" {
		err := repo.uploadLocalStorage(app.Cfg.FileStoragePath)
		if err != nil {
			repo.logger.Info("could not upload data from the file",
				zap.String("filepath", app.Cfg.FileStoragePath),
				zap.String("error", err.Error()))
			return nil, fmt.Errorf("could not upload data from the file: %w", err)
		}
		repo.logger.Info("successfully uploaded data from the file")

		return &repo, nil
	} else if repo.db == nil && app.Cfg.FileStoragePath == "" {
		repo.logger.Info("filepath for file storage is not set, keep working with in memory storage")
	}
	return &repo, nil
}

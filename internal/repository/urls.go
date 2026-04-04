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

type URLRepositoryInterface interface {
	GetList(ctx context.Context) ([]model.URLEntity, error)
	GetByShortURL(ctx context.Context, shortURL string) (string, error)
	Create(ctx context.Context, originalURL, shortURL string) (model.URLEntity, error)
	BulkCreate(ctx context.Context, urls []model.URLBulkCreate) ([]model.URLBulkCreate, error)
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
			return "", err
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
			return nil, err
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

	var result []model.URLBulkCreate

	if repo.db != nil {

		query := "INSERT INTO urls (short_url, original_url) VALUES ($1, $2)"
		stmt, err := repo.db.PrepareContext(ctx, query)
		if err != nil {
			return nil, err
		}

		tx, err := repo.db.BeginTxx(ctx, nil)
		if err != nil {
			return nil, err
		}
		for _, url := range urls {
			_, err = stmt.ExecContext(ctx, url.ShortURL, url.OriginalURL)
			if err != nil {
				tx.Rollback()
				return nil, err
			}
			entity := model.URLBulkCreate{
				CorrelationID: url.CorrelationID,
				ShortURL:      url.ShortURL,
				OriginalURL:   url.OriginalURL,
			}
			result = append(result, entity)

		}
		return result, tx.Commit()
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

func (repo *LocalURLRepository) Delete(ctx context.Context, shortURL string) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.db != nil {

		query := "DELETE FROM urls WHERE original_url = $1 RETURNING short_url"
		stmt, err := repo.db.PrepareContext(ctx, query)
		if err != nil {
			return fmt.Errorf("%w", err)
		}
		resp, err := stmt.ExecContext(ctx, shortURL)
		if resp != nil {
			return fmt.Errorf("%w", ErrURLNotFound)
		}
		if err != nil {
			return fmt.Errorf("%w", err)
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
			return false, err
		}
		result, err := stmt.QueryContext(ctx, shortURLList)
		if err != nil {
			return false, err
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

func (w *Writer) BulkWriteEntities(entities []model.URLBulkCreate) error {
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
		return nil, err
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
			return nil, err
		}
		result = append(result, object)
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
		return err
	}
	for _, object := range result {
		err := repo.SaveURLToLocalStorage(object)
		if err != nil {
			repo.logger.Info("could not upload URL Entity to local storage",
				zap.String("ShortURL", object.ShortURL),
				zap.String("OriginalURL", object.OriginalURL))
			return err
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
			return nil, err
		}
		repo.logger.Info("successfully uploaded data from the file")

		return &repo, nil
	} else if repo.db == nil && app.Cfg.FileStoragePath == "" {
		repo.logger.Info("filepath for file storage is not set, keep working with in memory storage")
	}
	return &repo, nil
}

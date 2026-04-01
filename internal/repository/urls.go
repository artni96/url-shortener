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
	"go.uber.org/zap"
)

var ErrURLAlreadyExists = errors.New("url already exists")
var ErrURLNotFound = errors.New("url not found")

type URLRepositoryInterface interface {
	GetByShortURL(ctx context.Context, shortURL string) (string, error)
	Create(ctx context.Context, originalURL, shortURL string) (model.URLEntity, error)
	SaveURL(ctx context.Context, url model.URLEntity) error
}
type LocalURLRepository struct {
	mu      sync.RWMutex
	urls    map[string]string
	app     *config.App
	storage string
}

func (repo *LocalURLRepository) Create(ctx context.Context, originalURL, shortURL string) (model.URLEntity, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.app.DB != nil {
		entity := model.URLEntity{}
		row := repo.app.DB.QueryRowContext(ctx, "SELECT * FROM urls WHERE short_url = $1", shortURL)
		err := row.Scan(&entity.OriginalURL, &entity.ShortURL)
		if err == nil {
			return model.URLEntity{}, err
		}

		_, err = repo.app.DB.ExecContext(ctx, "INSERT INTO urls (short_url, original_url) VALUES ($1, $2)", shortURL, originalURL)
		if err != nil {
			return model.URLEntity{}, err
		}
		entity.OriginalURL = originalURL
		entity.ShortURL = shortURL
		return entity, nil
	}
	if _, ok := repo.urls[originalURL]; ok {
		return model.URLEntity{}, ErrURLAlreadyExists
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

	if repo.app.DB != nil {
		row := repo.app.DB.QueryRowContext(ctx, "SELECT original_url FROM urls where short_url = $1", shortURL)
		err := row.Scan(&entity)
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

func (repo *LocalURLRepository) SaveURL(ctx context.Context, urlEntity model.URLEntity) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.app.DB != nil {

	}
	_, ok := repo.urls[urlEntity.ShortURL]
	if ok {
		return fmt.Errorf("%w: %s", ErrURLAlreadyExists, urlEntity.ShortURL)
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

func (w *Writer) WriteEntity(urlEntity *model.URLEntity) error {
	data, err := json.Marshal(&urlEntity)
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

func (repo *LocalURLRepository) uploadLocalStorage(ctx context.Context, filepath string, log *zap.Logger) error {
	fileReader, err := NewFileScanner(filepath)
	defer func(fileReader *FileScanner) {
		err := fileReader.Close()
		if err != nil {
			log.Error("could not close file reader",
				zap.String("error", err.Error()),
			)
		}
	}(fileReader)

	if err != nil {
		log.Error("could not initialize NewFileScanner",
			zap.String("error", err.Error()))
		return err
	}
	result, err := fileReader.CollectData()
	if err != nil {
		log.Error("could not collect data from file",
			zap.String("filepath", filepath),
			zap.String("error", err.Error()))
		return err
	}
	for _, object := range result {
		err := repo.SaveURL(ctx, object)
		if err != nil {
			log.Error("could not upload URL Entity to local storage",
				zap.String("ShortURL", object.ShortURL),
				zap.String("OriginalURL", object.OriginalURL))
			return err
		}
	}
	return nil
}

func NewURLRepository(ctx *context.Context, app *config.App, log *zap.Logger) (*LocalURLRepository, error) {
	repo := LocalURLRepository{urls: make(map[string]string), app: app}
	if app.DB == nil && app.Cfg.DatabaseDsn != "" {
		err := repo.uploadLocalStorage(*ctx, app.Cfg.FileStoragePath, log)
		if err != nil {
			log.Error("could not upload data from the file",
				zap.String("filepath", app.Cfg.FileStoragePath),
				zap.String("error", err.Error()))
			return nil, err
		}
	}
	return &repo, nil
}

package urls

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
	"go.uber.org/zap"
)

type InMemoryURLRepositoryInterface interface {
	Create(entity model.URLEntity) (model.URLEntity, error)
	BulkCreate(originalURLs []model.URLBulkCreate) ([]model.URLBulkCreate, error)
	GetByShortURL(shortURL string) (string, error)
	GetList() ([]model.URLEntity, error)
	Update(model.URLEntity) (model.URLEntity, error)
	Delete(shortURL string) error

	IsURLListUnique(shortURLList []string) (bool, error)
	UploadInMemoryStorage(url model.URLEntity) error
}
type InMemoryURLRepository struct {
	mu     sync.RWMutex
	urls   map[string]string
	logger *zap.Logger
}

func (repo *InMemoryURLRepository) Create(entity model.URLEntity) (model.URLEntity, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	if _, ok := repo.urls[entity.OriginalURL]; ok {
		return model.URLEntity{}, ErrShortURLAlreadyExists
	}

	for _, url := range repo.urls {
		if url == entity.OriginalURL {
			return entity, ErrOriginalURLAlreadyExists
		}
	}

	repo.urls[entity.ShortURL] = entity.OriginalURL
	return entity, nil
}

func (repo *InMemoryURLRepository) BulkCreate(urls []model.URLBulkCreate) ([]model.URLBulkCreate, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	var result []model.URLBulkCreate

	urlDuplicates := map[string]int{}
	var duplicateErrs []error

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

func (repo *InMemoryURLRepository) GetByShortURL(shortURL string) (string, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	var entity string

	entity, ok := repo.urls[shortURL]
	if !ok {
		return "", ErrURLNotFound
	}

	return entity, nil
}

func (repo *InMemoryURLRepository) GetList() ([]model.URLEntity, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	var entities []model.URLEntity

	for shortURL, originalURL := range repo.urls {
		entities = append(entities, model.URLEntity{OriginalURL: originalURL, ShortURL: shortURL})
	}
	return entities, nil
}

func (repo *InMemoryURLRepository) Update(entity model.URLEntity) (model.URLEntity, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	updatedEntity := model.URLEntity{}

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

func (repo *InMemoryURLRepository) Delete(shortURL string) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	for url := range repo.urls {
		if url == shortURL {
			delete(repo.urls, shortURL)
			return nil
		}
	}
	return fmt.Errorf("%w", ErrURLNotFound)
}

func (repo *InMemoryURLRepository) IsURLListUnique(shortURLList []string) (bool, error) {
	for _, shortURL := range shortURLList {
		if _, ok := repo.urls[shortURL]; ok {
			return false, ErrURLListIsNotUnique
		}
	}
	return true, nil
}

func (repo *InMemoryURLRepository) UploadInMemoryStorage(urlEntity model.URLEntity) error {
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

func (repo *InMemoryURLRepository) uploadInMemoryStorage(filepath string) error {
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
		err := repo.UploadInMemoryStorage(object)
		if err != nil {
			repo.logger.Info("could not upload URL Entity to local storage",
				zap.String("ShortURL", object.ShortURL),
				zap.String("OriginalURL", object.OriginalURL))
			return fmt.Errorf("could not upload URL Entity to local storage: %w", err)
		}
	}
	return nil
}

func NewInMemoryURLRepository(app *config.App) (*InMemoryURLRepository, error) {
	repo := InMemoryURLRepository{
		urls:   make(map[string]string),
		logger: app.Logger,
	}

	err := repo.uploadInMemoryStorage(app.Cfg.FileStoragePath)
	if err != nil {
		repo.logger.Info("could not upload data from the file",
			zap.String("filepath", app.Cfg.FileStoragePath),
			zap.String("error", err.Error()))
		return nil, fmt.Errorf("could not upload data from the file: %w", err)
	}
	repo.logger.Info("successfully uploaded data from the file")

	return &repo, nil

}

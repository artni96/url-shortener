package urls

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"go.uber.org/zap"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
)

// InMemoryURLRepositoryInterface encapsulates the logic to handle URL entities via a file.
type InMemoryURLRepositoryInterface interface {
	// Create saves a new URL entity.
	Create(entity model.URLCreate) (model.URLEntity, error)
	// BulkCreate saves several URL entities by one commit.
	BulkCreate(originalURLs []model.URLBulkCreate) ([]model.URLBulkCreate, error)
	// GetByShortURL returns a URL entity with original URL value by its short URL.
	GetByShortURL(shortURL string) (model.GetByShortURLResponse, error)
	// GetList returns the list of all URL entities.
	GetList() ([]model.URLEntity, error)
	// GetUserList returns the list of user's URL entities.
	GetUserList(createdBy int) ([]model.URLEntity, error)
	// Update saves the changes of a URL entity gotten by its short URL.
	Update(model.URLEntity) (model.URLEntity, error)
	// Delete removes a URL entity by its short URL value.
	Delete(shortURL string) error
	// BulkDelete removes several URL entities by one commit.
	BulkDelete(urls []model.URLDelete) ([]error, error)

	// IsURLListUnique checks whether the whole list of generated short URL values is unique.
	IsURLListUnique(shortURLList []string) (bool, error)
	UploadInMemoryStorage(url model.URLEntity) error
}
type InMemoryURLRepository struct {
	mu     sync.RWMutex
	urls   map[string]model.URLNestedData
	logger *zap.Logger
}

// Create saves a new URL entity.
func (repo *InMemoryURLRepository) Create(entity model.URLCreate) (model.URLEntity, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	responseEntity := model.URLEntity{}

	if _, ok := repo.urls[entity.OriginalURL]; ok {
		return responseEntity, ErrShortURLAlreadyExists
	}

	responseEntity.ShortURL = entity.ShortURL
	responseEntity.OriginalURL = entity.OriginalURL
	responseEntity.CreatedBy = entity.CreatedBy
	for _, url := range repo.urls {
		if url.OriginalURL == entity.OriginalURL {
			return responseEntity, ErrOriginalURLAlreadyExists
		}
	}

	repo.urls[entity.ShortURL] = model.URLNestedData{
		OriginalURL: entity.OriginalURL,
		CreatedBy:   entity.CreatedBy,
	}
	return responseEntity, nil
}

// BulkCreate saves several URL entities by one commit.
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
		repo.urls[url.ShortURL] = model.URLNestedData{
			OriginalURL: url.OriginalURL,
			CreatedBy:   url.CreatedBy,
		}
		entity := model.URLBulkCreate{
			CorrelationID: url.CorrelationID,
			ShortURL:      url.ShortURL,
			OriginalURL:   url.OriginalURL,
			CreatedBy:     url.CreatedBy,
		}
		result = append(result, entity)
	}

	return result, nil
}

// GetByShortURL returns a URL entity with original URL value by its short URL.
func (repo *InMemoryURLRepository) GetByShortURL(shortURL string) (model.GetByShortURLResponse, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	var entity model.URLNestedData

	entity, ok := repo.urls[shortURL]
	if !ok {
		return model.GetByShortURLResponse{}, ErrURLNotFound
	}

	return model.GetByShortURLResponse{
		OriginalURL: entity.OriginalURL,
		ShortURL:    shortURL,
		IsDeleted:   entity.IsDeleted,
	}, nil
}

// GetList returns the list of all URL entities.
func (repo *InMemoryURLRepository) GetList() ([]model.URLEntity, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	var entities []model.URLEntity

	for shortURL, data := range repo.urls {
		entities = append(entities, model.URLEntity{OriginalURL: data.OriginalURL, ShortURL: shortURL, CreatedBy: data.CreatedBy, IsDeleted: data.IsDeleted})
	}
	return entities, nil
}

// GetUserList returns the list of user's URL entities.
func (repo *InMemoryURLRepository) GetUserList(createdBy int) ([]model.URLEntity, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	var entities []model.URLEntity

	for shortURL, data := range repo.urls {
		if data.CreatedBy == createdBy {
			entities = append(entities, model.URLEntity{OriginalURL: data.OriginalURL, ShortURL: shortURL, CreatedBy: data.CreatedBy, IsDeleted: data.IsDeleted})
		}
	}
	return entities, nil
}

// Update saves the changes of a URL entity gotten by its short URL.
func (repo *InMemoryURLRepository) Update(entity model.URLEntity) (model.URLEntity, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	updatedEntity := model.URLEntity{}

	for _, data := range repo.urls {
		if data.OriginalURL == entity.OriginalURL {
			return updatedEntity, ErrOriginalURLAlreadyExists
		}
	}

	for dbShortURL := range repo.urls {
		if dbShortURL == entity.ShortURL {
			repo.urls[dbShortURL] = model.URLNestedData{
				OriginalURL: entity.OriginalURL,
				CreatedBy:   -1,
			}
			updatedEntity.OriginalURL = entity.OriginalURL
			updatedEntity.ShortURL = entity.ShortURL
			return updatedEntity, nil
		}
	}
	return updatedEntity, ErrURLNotFound
}

// Delete removes a URL entity by its short URL value.
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

// BulkDelete removes several URL entities by one commit.
func (repo *InMemoryURLRepository) BulkDelete(urls []model.URLDelete) ([]error, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	var errs []error

	for _, url := range urls {
		urlData, ok := repo.urls[url.ShortURL]
		if !ok {
			errs = append(errs, fmt.Errorf("%w, short url: %s", ErrURLNotFound, url.ShortURL))
		} else if urlData.CreatedBy != url.CreatedBy {
			errs = append(errs, fmt.Errorf("%w: url author id: %d, request user id: %d", ErrUserIsNotAuthor, urlData.CreatedBy, url.CreatedBy))
		} else {
			urlData.IsDeleted = true
			repo.urls[url.ShortURL] = urlData
		}
	}

	return errs, nil
}

// IsURLListUnique checks whether the whole list of generated short URL values is unique.
func (repo *InMemoryURLRepository) IsURLListUnique(shortURLList []string) (bool, error) {
	for _, shortURL := range shortURLList {
		if _, ok := repo.urls[shortURL]; ok {
			return false, ErrURLListIsNotUnique
		}
	}
	return true, nil
}

// UploadInMemoryStorage saves a URL entity to the in-memory storage from the file.
func (repo *InMemoryURLRepository) UploadInMemoryStorage(urlEntity model.URLEntity) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	_, ok := repo.urls[urlEntity.ShortURL]
	if ok {
		return fmt.Errorf("%w: %s", ErrShortURLAlreadyExists, urlEntity.ShortURL)
	}
	repo.urls[urlEntity.ShortURL] = model.URLNestedData{
		OriginalURL: urlEntity.OriginalURL,
		CreatedBy:   urlEntity.CreatedBy,
		IsDeleted:   urlEntity.IsDeleted,
	}
	return nil
}

// Writer is a custom writer to write new Entity data into the file.
type Writer struct {
	file   *os.File
	writer *bufio.Writer
}

// NewWriter initializes a new Writer object.
func NewWriter(filename string) (*Writer, error) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("could open file: %w", err)
	}
	return &Writer{file: file, writer: bufio.NewWriter(file)}, nil
}

// WriteEntity saves URL entity value into the file.
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

// BulkWriteEntities saves multiple URL entities values into the file.
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
			CreatedBy:   entity.CreatedBy,
			IsDeleted:   entity.IsDeleted,
		}
		data, err := json.Marshal(&entityForFile)
		if err != nil {
			return fmt.Errorf("could not marshal entity: %w", err)
		}

		if _, err = w.writer.Write(data); err != nil {
			return fmt.Errorf("could not write urls data to file: %w", err)
		}

		if err = w.writer.WriteByte('\n'); err != nil {
			return fmt.Errorf("could not write urls data to file: %w", err)
		}
		defer w.Close()
	}
	return w.writer.Flush()
}

// Close finishes Writer interaction with the file.
func (w *Writer) Close() error {
	return w.file.Close()
}

// FileScanner is a custom reader to get URL data from the file.
type FileScanner struct {
	file    *os.File
	scanner *bufio.Scanner
}

// NewFileScanner initializes a new FileScanner.
func NewFileScanner(filename string) (*FileScanner, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	return &FileScanner{file: file, scanner: bufio.NewScanner(file)}, nil
}

// Close finishes FileScanner interaction with the file.
func (s *FileScanner) Close() error {
	return s.file.Close()
}

// CollectData collects URL data from the file.
func (s *FileScanner) CollectData() ([]model.URLEntity, error) {
	var result []model.URLEntity
	for s.scanner.Scan() {

		data := s.scanner.Bytes()

		object := model.URLEntity{}
		if err := json.Unmarshal(data, &object); err != nil {
			return nil, fmt.Errorf("could not unmarshal object: %w", err)
		}
		if object.ShortURL != "" && object.OriginalURL != "" {
			result = append(result, object)
		}

	}
	return result, nil
}

// CollectFilteredData returns a URL entity from the file by its short URL.
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

// uploadInMemoryStorage fills up the in-memory storage with URL values from the file.
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

// NewInMemoryURLRepository initializes a new InMemoryURLRepository.
func NewInMemoryURLRepository(app *config.App) (*InMemoryURLRepository, error) {
	repo := InMemoryURLRepository{
		urls:   make(map[string]model.URLNestedData),
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

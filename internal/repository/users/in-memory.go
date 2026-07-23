package users

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

// InMemoryUserRepositoryInterface encapsulates the logic to handle User entities via in-memory storage.
type InMemoryUserRepositoryInterface interface {
	// Create saves a new User entity in the in-memory storage by user's IP.
	Create(ip string) (model.User, error)
	// GetByIP returns a User data from the in-memory storage by its ID.
	GetByIP(ip string) (int, error)
	// GetStats provides the number of unique users in the in-memory storage,
	GetStats() int64

	uploadInMemoryStorage(filepath string) error
}

// InMemoryUserRepository implements an object-medicator with the in-memory storage.
type InMemoryUserRepository struct {
	mu     sync.RWMutex
	users  map[int]string
	logger *zap.Logger
}

// Create saves a new User entity in the in-memory storage by user's IP.
func (repo *InMemoryUserRepository) Create(ip string) (model.User, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	var lastUserID int
	for id := range repo.users {
		if id > lastUserID {
			lastUserID = id
		}
	}
	userID := lastUserID + 1
	repo.users[userID] = ip
	return model.User{ID: userID, IP: ip}, nil
}

// GetByIP returns a User data from the in-memory storage by its ID.
func (repo *InMemoryUserRepository) GetByIP(ip string) (int, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	if len(repo.users) == 0 {
		return -1, errors.New("user not found")
	}
	for userID, userIP := range repo.users {
		if userIP == ip {
			return userID, nil
		}
	}
	return -1, fmt.Errorf("user not found for ip %s", ip)
}

// GetStats provides the number of unique users in the in-memory storage.
func (repo *InMemoryUserRepository) GetStats() int64 {
	return int64(len(repo.users))
}

// NewInMemoryUserRepository implements a new InMemoryUserRepository.
func NewInMemoryUserRepository(cfg *config.Config, logger *zap.Logger) (*InMemoryUserRepository, error) {
	repo := InMemoryUserRepository{
		users:  make(map[int]string),
		logger: logger,
	}

	err := repo.uploadInMemoryStorage(cfg.FileStoragePath)
	if err != nil {
		repo.logger.Info("could not upload data from the file",
			zap.String("filepath", cfg.FileStoragePath),
			zap.String("error", err.Error()))
		return nil, fmt.Errorf("could not upload data from the file: %w", err)
	}
	repo.logger.Info("successfully uploaded data from the file")
	return &repo, nil
}

// Writer is a custom writer to write new User data into the file.
type Writer struct {
	file   *os.File
	writer *bufio.Writer
}

// NewWriter initializes a new Writer.
func NewWriter(filename string) (*Writer, error) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("could open file: %w", err)
	}
	return &Writer{file: file, writer: bufio.NewWriter(file)}, nil
}

// WriteEntity saves a new User entity into the file.
func (w *Writer) WriteEntity(user model.User) error {
	data, err := json.Marshal(&user)
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

// BulkWriteEntities saves several new User entities into the file.
func (w *Writer) BulkWriteEntities(entities []model.User, toTruncate bool) error {
	if toTruncate {
		err := os.Truncate(w.file.Name(), 0)
		if err != nil {
			return fmt.Errorf("could not truncate file at users removal: %w", err)
		}
	}

	defer w.Close()
	for _, entity := range entities {
		entityForFile := model.User{
			ID: entity.ID,
			IP: entity.IP,
		}
		data, err := json.Marshal(&entityForFile)
		if err != nil {
			return fmt.Errorf("could not marshal entity: %w", err)
		}

		if _, err = w.writer.Write(data); err != nil {
			return fmt.Errorf("could not write users data to file: %w", err)
		}

		if err = w.writer.WriteByte('\n'); err != nil {
			return fmt.Errorf("could not write users data to file: %w", err)
		}
		defer w.Close()
	}
	return w.writer.Flush()
}

// Close gently finishes Writer interaction with the file.
func (w *Writer) Close() error {
	return w.file.Close()
}

// FileScanner is a custom reader to get User data from the file.
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

// Close gently finishes FileScanner interaction with the file.
func (s *FileScanner) Close() error {
	return s.file.Close()
}

// CollectData extracts User entities from the file.
func (s *FileScanner) CollectData() ([]model.User, error) {
	var result []model.User
	for s.scanner.Scan() {

		data := s.scanner.Bytes()

		var user model.User
		if err := json.Unmarshal(data, &user); err != nil {
			return nil, fmt.Errorf("could not unmarshal object: %w", err)
		}
		if user.IP != "" && user.ID != 0 {
			result = append(result, user)
		}
	}
	return result, nil
}

// UploadInMemoryStorage saves a new User entity from the file in the in-memory storage.
func (repo *InMemoryUserRepository) UploadInMemoryStorage(user model.User) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	repo.users[user.ID] = user.IP
	return nil
}

// uploadInMemoryStorage updates the in-memory storage with data from the file.
func (repo *InMemoryUserRepository) uploadInMemoryStorage(filepath string) error {
	fileReader, err := NewFileScanner(filepath)
	if fileReader == nil {
		return nil
	}
	defer func(fileReader *FileScanner) {
		err = fileReader.Close()
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
	for _, user := range result {
		err = repo.UploadInMemoryStorage(user)
		if err != nil {
			repo.logger.Info("could not upload User Entity to local storage",
				zap.Int("user ID", user.ID),
				zap.String("user IP", user.IP),
			)
			return fmt.Errorf("could not upload User Entity to local storage: %w", err)
		}
	}
	return nil
}

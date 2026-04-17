package users

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

type InMemoryUserRepositoryInterface interface {
	Create(ip string) (model.User, error)
	GetByIP(ip string) (int, error)

	uploadInMemoryStorage(filepath string) error
}

type InMemoryUserRepository struct {
	mu     sync.RWMutex
	users  map[int]string
	logger *zap.Logger
}

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

func NewInMemoryUserRepository(app *config.App) (*InMemoryUserRepository, error) {
	repo := InMemoryUserRepository{
		users:  make(map[int]string),
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

func (s *FileScanner) CollectData() ([]model.User, error) {
	var result []model.User
	for s.scanner.Scan() {

		data := s.scanner.Bytes()

		var user model.User
		if err := json.Unmarshal(data, &user); err != nil {
			return nil, fmt.Errorf("could not unmarshal object: %w", err)
		}

	}
	return result, nil
}

func (repo *InMemoryUserRepository) UploadInMemoryStorage(user model.User) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	repo.users[user.ID] = user.IP
	return nil
}

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

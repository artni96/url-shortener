package users

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/artni96/url-shortener/internal/config"
	"go.uber.org/zap"
)

type InMemoryUserRepositoryInterface interface {
	Create() (int, error)

	uploadInMemoryStorage(filepath string) error
}

type InMemoryUserRepository struct {
	mu     sync.Mutex
	users  []int
	logger *zap.Logger
}

func (repo *InMemoryUserRepository) Create() (int, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	var lastUserID int
	for id := range repo.users {
		if id > lastUserID {
			lastUserID = id
		}
	}
	userID := lastUserID + 1

	return userID, nil
}

func NewInMemoryUserRepository(app *config.App) (*InMemoryUserRepository, error) {
	repo := InMemoryUserRepository{
		users:  []int{},
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

func (w *Writer) WriteEntity(userID int) error {
	data, err := json.Marshal(&userID)
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

func (s *FileScanner) CollectData() ([]int, error) {
	var result []int
	for s.scanner.Scan() {

		data := s.scanner.Bytes()

		var userID int
		if err := json.Unmarshal(data, &userID); err != nil {
			return nil, fmt.Errorf("could not unmarshal object: %w", err)
		}

	}
	return result, nil
}

func (repo *InMemoryUserRepository) UploadInMemoryStorage(userID int) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	//_, ok := repo.users[user.ID]
	//if ok {
	//	return fmt.Errorf("%w: user id - %d", ErrUserAlreadyExists, user.ID)
	//}
	//repo.users[user.ID] = map[string]string{}
	//repo.users[user.ID]["username"] = user.Username
	//repo.users[user.ID]["password"] = user.Password
	repo.users = append(repo.users, userID)
	return nil
}

func (repo *InMemoryUserRepository) uploadInMemoryStorage(filepath string) error {
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
	for _, userID := range result {
		err := repo.UploadInMemoryStorage(userID)
		if err != nil {
			repo.logger.Info("could not upload User Entity to local storage",
				zap.Int("user ID", userID),
			)
			return fmt.Errorf("could not upload User Entity to local storage: %w", err)
		}
	}
	return nil
}

package auth

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
	"go.uber.org/zap"
)

type InMemoryAuthRepositoryInterface interface {
	Create(user model.UserCreate) (model.UserWithHashedPassword, error)
	GetUserHashedPassword(username string) (model.UserWithHashedPassword, error)

	uploadInMemoryStorage(filepath string) error
}

type InMemoryAuthRepository struct {
	mu     sync.Mutex
	users  map[int]map[string]string
	logger *zap.Logger
}

func (repo *InMemoryAuthRepository) Create(user model.UserCreate) (model.UserWithHashedPassword, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	responseEntity := model.UserWithHashedPassword{}

	var lastUserID int
	for id, value := range repo.users {
		if id > lastUserID {
			lastUserID = id
		}
		if value["username"] == user.Username {
			return responseEntity, ErrUserAlreadyExists
		}
	}
	entityID := lastUserID + 1
	repo.users[entityID] = map[string]string{}
	repo.users[entityID]["username"] = user.Username
	repo.users[entityID]["password"] = user.HashedPassword

	responseEntity.ID = entityID
	responseEntity.Username = user.Username
	responseEntity.Password = user.HashedPassword
	return responseEntity, nil
}

func (repo *InMemoryAuthRepository) GetUserHashedPassword(username string) (model.UserWithHashedPassword, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	userResponse := model.UserWithHashedPassword{}

	for id, user := range repo.users {
		if user["username"] == username {
			userResponse.ID = id
			userResponse.Username = user["username"]
			userResponse.Password = user["password"]

			return userResponse, nil
		}
	}
	return userResponse, ErrUserNotFound
}

func NewInMemoryAuthRepository(app *config.App) (*InMemoryAuthRepository, error) {
	repo := InMemoryAuthRepository{
		users:  make(map[int]map[string]string),
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

func (w *Writer) WriteEntity(entity *model.UserWithHashedPassword) error {
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

func (s *FileScanner) CollectData() ([]model.UserWithHashedPassword, error) {
	var result []model.UserWithHashedPassword
	for s.scanner.Scan() {

		data := s.scanner.Bytes()

		object := model.UserWithHashedPassword{}
		if err := json.Unmarshal(data, &object); err != nil {
			return nil, fmt.Errorf("could not unmarshal object: %w", err)
		}
		if object.Username != "" && object.Password != "" {
			result = append(result, object)
		}

	}
	return result, nil
}

func (repo *InMemoryAuthRepository) UploadInMemoryStorage(user model.UserWithHashedPassword) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	_, ok := repo.users[user.ID]
	if ok {
		return fmt.Errorf("%w: user id - %d", ErrUserAlreadyExists, user.ID)
	}
	repo.users[user.ID] = map[string]string{}
	repo.users[user.ID]["username"] = user.Username
	repo.users[user.ID]["password"] = user.Password
	return nil
}

func (repo *InMemoryAuthRepository) uploadInMemoryStorage(filepath string) error {
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
			repo.logger.Info("could not upload User Entity to local storage",
				zap.Int("ID", object.ID),
				zap.String("Username", object.Username))
			return fmt.Errorf("could not upload User Entity to local storage: %w", err)
		}
	}
	return nil
}

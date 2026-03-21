package service

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/logger"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/repository"
	"github.com/artni96/url-shortener/internal/utility"
)

var ErrFailedToCreated = errors.New("could not create ShortURL")

type URLServiceInterface interface {
	GetByID(urlID string) (string, error)
	Create(urlStr string) (string, error)
}
type URLService struct {
	repo repository.URLRepositoryInterface
	cfg  *config.Config
}

func (s *URLService) GetByID(urlID string) (string, error) {
	resp, err := s.repo.GetByID(urlID)
	if err != nil {
		return "", err
	}
	return resp.OriginalURL, nil
}

func (s *URLService) Create(urlStr string) (string, error) {

	for i := range 5 {
		urlID, err := utility.GenerateID(10)
		if err != nil {
			logger.Logger.Infof("urlID creation, attempt №%d\n", i)
			continue
		}
		resp, err := s.repo.Create(urlStr, urlID)
		if err != nil {
			if errors.Is(err, repository.ErrURLAlreadyExists) {
				logger.Logger.Infof("urlID creation, attempt №%d\n", i)
				continue
			} else {
				logger.Logger.Errorf("could not manage to create a short url for %s\n", urlStr)
				return "", err
			}
		}
		if s.cfg.Mode != "test" {
			fileWriter, err := NewWriter(s.cfg.FileStoragePath)
			if err != nil {
				return "", err
			}
			defer fileWriter.Close()

			err = fileWriter.WriteEntity(&resp)
			if err != nil {
				return "", err
			}
		}
		return resp.ShortURL, nil
	}
	return "", fmt.Errorf("%w %s", ErrFailedToCreated, urlStr)
}

func NewURLService(repo repository.URLRepositoryInterface, cfg *config.Config) *URLService {
	return &URLService{
		repo: repo, cfg: cfg,
	}
}

type Writer struct {
	file   *os.File
	writer *bufio.Writer
}

func NewWriter(filename string) (*Writer, error) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &Writer{file: file, writer: bufio.NewWriter(file)}, nil
}

func (w *Writer) WriteEntity(urlEntity *model.URLEntity) error {
	data, err := json.Marshal(&urlEntity)
	if err != nil {
		return err
	}

	if _, err := w.writer.Write(data); err != nil {
		return err
	}

	if ere := w.writer.WriteByte('\n'); ere != nil {
		return err
	}
	return w.writer.Flush()
}

func (w *Writer) Close() error {
	return w.file.Close()
}

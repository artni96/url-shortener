package service

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/repository"
	"github.com/artni96/url-shortener/internal/utility"
	"go.uber.org/zap"
)

var ErrFailedToCreated = errors.New("could not create ShortURL")

type URLServiceInterface interface {
	GetByID(urlID string) (string, error)
	Create(urlStr string) (string, error)
}
type URLService struct {
	repo repository.URLRepositoryInterface
	cfg  *config.Config
	log  *zap.Logger
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
			s.log.Info(
				"urlID creation",
				zap.Int("attempt №", i),
			)
			continue
		}
		resp, err := s.repo.Create(urlStr, urlID)
		if err != nil {
			if errors.Is(err, repository.ErrURLAlreadyExists) {
				s.log.Info(
					"urlID creation",
					zap.Int("attempt №", i),
				)
				continue
			} else {
				s.log.Error(
					"could not manage to create a short url for ",
					zap.String("url", urlStr),
				)
				return "", err
			}
		}
		if s.cfg.Mode != "test" {
			fileWriter, err := NewWriter(s.cfg.FileStoragePath)
			if err != nil {
				s.log.Error("could not create file writer",
					zap.String("path", s.cfg.FileStoragePath),
					zap.String("error message", err.Error()),
				)
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
	return w.writer.Flush()
}

func (w *Writer) Close() error {
	return w.file.Close()
}

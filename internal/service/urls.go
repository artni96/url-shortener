package service

import (
	"bufio"
	"errors"
	"fmt"
	"os"

	"github.com/artni96/url-shortener/internal/logger"
	"github.com/artni96/url-shortener/internal/repository"
	"github.com/artni96/url-shortener/internal/utility"
)

var ErrFailedToCreated = errors.New("не удалось создать ссылку для")

type URLServiceInterface interface {
	Get(urlID string) (string, error)
	Create(urlStr string) (string, error)
}
type URLService struct {
	repo repository.URLRepositoryInterface
}

func (s *URLService) Get(urlID string) (string, error) {
	resp, err := s.repo.GetByID(urlID)
	if err != nil {
		return "", err
	}
	return resp, nil
}

func (s *URLService) Create(urlStr string) (string, error) {

	for i := range 5 {
		urlID, err := utility.GenerateID(10)
		if err != nil {
			logger.Logger.Infof("создание urlID, попытка №%d\n", i)
			continue
		}
		resp, err := s.repo.Create(urlStr, urlID)
		if err != nil {
			if errors.Is(err, repository.ErrURLIDDuplicate) {
				logger.Logger.Infof("создание urlID, попытка №%d\n", i)
				continue
			} else {
				logger.Logger.Errorf("не удалось создать короткую ссылку для %s\n", urlStr)
				return "", err
			}
		}
		return resp.ShortURL, nil
	}
	return "", fmt.Errorf("%w %s", ErrFailedToCreated, urlStr)
}

func NewURLService(repo repository.URLRepositoryInterface) *URLService {
	return &URLService{repo: repo}
}

type Writer struct {
	file    *os.File
	scanner *bufio.Writer
}

func NewWriter(filename string) (*Writer, error) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &Writer{file: file, scanner: bufio.NewWriter(file)}, nil
}

func (w *Writer) WriteObject(urlStr string) error {
	return nil
}

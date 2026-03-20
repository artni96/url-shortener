package service

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/artni96/url-shortener/internal/logger"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/repository"
	"github.com/artni96/url-shortener/internal/utility"
)

var ErrFailedToCreated = errors.New("не удалось создать ссылку для")

type URLServiceInterface interface {
	GetByID(urlID string) (string, error)
	Create(urlStr string) (string, error)
}
type URLService struct {
	repo repository.URLRepositoryInterface
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
		filename := "test.json"
		fileWriter, err := NewWriter(filename)
		if err != nil {
			return "", err
		}
		defer fileWriter.Close()

		err = fileWriter.WriteObject(&resp)
		if err != nil {
			return "", err
		}
		return resp.ShortURL, nil
	}
	return "", fmt.Errorf("%w %s", ErrFailedToCreated, urlStr)
}

func NewURLService(repo repository.URLRepositoryInterface) *URLService {
	return &URLService{repo: repo}
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

func (w *Writer) WriteObject(urlEntity *model.URLEntity) error {
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

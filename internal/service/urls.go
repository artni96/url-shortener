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

var ErrFailedToCreated = errors.New("не удалось создать ссылку для")

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
			if errors.Is(err, repository.ErrURLIDDuplicate) {
				logger.Logger.Infof("urlID creation, attempt №%d\n", i)
				continue
			} else {
				logger.Logger.Errorf("не удалось создать короткую ссылку для %s\n", urlStr)
				return "", err
			}
		}
		fileWriter, err := NewWriter(s.cfg.FileStoragePath)
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

func NewURLService(repo repository.URLRepositoryInterface, cfg config.Config) *URLService {
	return &URLService{repo: repo, cfg: &cfg}
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

type Reader struct {
	file    *os.File
	scanner *bufio.Scanner
}

func NewReader(filename string) (*Reader, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	return &Reader{file: file, scanner: bufio.NewScanner(file)}, nil
}

func (r *Reader) Close() error {
	return r.file.Close()
}

func (r *Reader) ReadAllObjects() ([]model.URLEntity, error) {
	var result []model.URLEntity
	for r.scanner.Scan() {

		data := r.scanner.Bytes()

		object := model.URLEntity{}
		if err := json.Unmarshal(data, &object); err != nil {
			return nil, err
		}
		result = append(result, object)
	}
	return result, nil
}

func (s *URLService) BulkCreate(filepath string) error {
	fileReader, err := NewReader(filepath)
	defer func(fileReader *Reader) {
		err := fileReader.Close()
		if err != nil {

		}
	}(fileReader)

	if err != nil {
		return err
	}
	result, err := fileReader.ReadAllObjects()
	if err != nil {
		return err
	}
	for _, object := range result {
		err := s.repo.UploadURL(object)
		if err != nil {
			return err
		}
	}

	return nil
}

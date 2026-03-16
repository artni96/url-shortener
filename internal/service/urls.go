package service

import (
	"errors"
	"fmt"

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
	resp, err := s.repo.Get(urlID)
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
		return resp, nil
	}
	return "", fmt.Errorf("%w %s", ErrFailedToCreated, urlStr)
}

func NewURLService(repo repository.URLRepositoryInterface) *URLService {
	return &URLService{repo: repo}
}

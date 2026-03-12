package service

import (
	"errors"
	"fmt"

	"github.com/artni96/url-shortener/internal/repository"
	"github.com/artni96/url-shortener/internal/utility"
)

var ErrFailedToCreated = errors.New("Не удалось создать ссылку для")

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
	for range 5 {
		urlID, err := utility.GenerateID(10)
		if err != nil {
			continue
		}
		resp, err := s.repo.Create(urlStr, urlID)
		if err != nil {
			if errors.Is(err, repository.ErrURLIDDuplicate) {
				continue
			} else {
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

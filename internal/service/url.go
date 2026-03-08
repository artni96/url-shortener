package service

import (
	"github.com/artni96/url-shortener/internal/repository"
)

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

func (s *URLService) Create(urlStr, urlID string) (string, error) {
	resp, err := s.repo.Create(urlStr, urlID)
	if err != nil {
		return "", err
	}
	return resp, nil
}

type URLServiceInterface interface {
	Get(urlID string) (string, error)
	Create(urlStr, urlID string) (string, error)
}

func NewURLService(repo repository.URLRepositoryInterface) *URLService {
	return &URLService{repo: repo}
}

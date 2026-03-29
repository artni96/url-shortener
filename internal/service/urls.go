package service

import (
	"errors"
	"fmt"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/repository"
	"github.com/artni96/url-shortener/internal/utility"
	"go.uber.org/zap"
)

var ErrFailedToCreated = errors.New("could not create ShortURL")

type URLServiceInterface interface {
	GetByShortURL(urlID string) (string, error)
	Create(urlStr string) (string, error)
}
type URLService struct {
	repo repository.URLRepositoryInterface
	cfg  *config.Config
	log  *zap.Logger
}

func (s *URLService) GetByShortURL(shortURL string) (string, error) {
	originalURL, err := s.repo.GetByShortURL(shortURL)
	if err != nil {
		return "", err
	}
	return originalURL, nil
}

func (s *URLService) Create(originalURL string) (string, error) {

	for i := range 5 {
		urlID, err := utility.GenerateShortURL(10)
		if err != nil {
			s.log.Info(
				"urlID creation",
				zap.Int("attempt №", i),
			)
			continue
		}
		entity, err := s.repo.Create(originalURL, urlID)
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
					zap.String("url", originalURL),
				)
				return "", err
			}
		}

		if s.cfg.FileStoragePath != "" {
			fileWriter, err := repository.NewWriter(s.cfg.FileStoragePath)
			if err != nil {
				s.log.Error("could not create file writer",
					zap.String("path", s.cfg.FileStoragePath),
					zap.String("error message", err.Error()),
				)
				return "", err
			}
			defer fileWriter.Close()

			err = fileWriter.WriteEntity(&entity)
			if err != nil {
				return "", err
			}
		}

		return entity.ShortURL, nil
	}
	return "", fmt.Errorf("%w %s", ErrFailedToCreated, originalURL)
}

func NewURLService(repo repository.URLRepositoryInterface, cfg *config.Config) *URLService {
	return &URLService{
		repo: repo, cfg: cfg,
	}
}

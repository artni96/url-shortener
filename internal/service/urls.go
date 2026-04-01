package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/repository"
	"github.com/artni96/url-shortener/internal/utility"
	"go.uber.org/zap"
)

var ErrFailedToCreated = errors.New("could not create ShortURL")

type URLServiceInterface interface {
	GetByShortURL(ctx context.Context, urlID string) (string, error)
	Create(ctx context.Context, urlStr string) (string, error)
}
type URLService struct {
	repo repository.URLRepositoryInterface
	log  *zap.Logger
	app  *config.App
}

func (s *URLService) GetByShortURL(ctx context.Context, shortURL string) (string, error) {
	originalURL, err := s.repo.GetByShortURL(ctx, shortURL)
	if err != nil {
		return "", err
	}
	return originalURL, nil
}

func (s *URLService) Create(ctx context.Context, originalURL string) (string, error) {

	for i := range 5 {
		urlID, err := utility.GenerateShortURL(10)
		if err != nil {
			s.log.Info(
				"urlID creation",
				zap.Int("attempt №", i),
			)
			continue
		}
		entity, err := s.repo.Create(ctx, originalURL, urlID)
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

		if s.app.Cfg.FileStoragePath != "" {
			fileWriter, err := repository.NewWriter(s.app.Cfg.FileStoragePath)
			if err != nil {
				s.log.Error("could not create file writer",
					zap.String("path", s.app.Cfg.FileStoragePath),
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

func NewURLService(repo repository.URLRepositoryInterface, app *config.App) *URLService {
	return &URLService{
		repo: repo, app: app,
	}
}

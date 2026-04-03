package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/repository"
	"github.com/artni96/url-shortener/internal/utility"
	"go.uber.org/zap"
)

var ErrFailedToCreated = errors.New("could not create ShortURL")

type URLServiceInterface interface {
	GetList(ctx context.Context) ([]model.URLEntity, error)
	GetByShortURL(ctx context.Context, urlID string) (string, error)
	Create(ctx context.Context, urlStr string, responseDomain string) (string, error)
	BulkCreate(ctx context.Context, urls []model.URLBulkCreateRequest, responseDomain string) ([]model.URLBulkCreateResponse, error)
}
type URLService struct {
	repo repository.URLRepositoryInterface
	log  *zap.Logger
	app  *config.App
}

func (s *URLService) GetList(ctx context.Context) ([]model.URLEntity, error) {
	result, err := s.repo.GetList(ctx)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *URLService) GetByShortURL(ctx context.Context, shortURL string) (string, error) {
	originalURL, err := s.repo.GetByShortURL(ctx, shortURL)
	if err != nil {
		return "", err
	}
	return originalURL, nil
}

func (s *URLService) Create(ctx context.Context, originalURL string, responseDomain string) (string, error) {

	for i := range 5 {
		shortURL, err := utility.GenerateShortURL(10)
		if err != nil {
			s.log.Info(
				"urlID creation",
				zap.Int("attempt №", i),
			)
			continue
		}
		entity, err := s.repo.Create(ctx, originalURL, shortURL)
		if err != nil {
			if errors.Is(err, repository.ErrURLAlreadyExists) {
				s.log.Info(
					"shortURL creation",
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

		return responseDomain + "/" + entity.ShortURL, nil
	}
	return "", fmt.Errorf("%w %s", ErrFailedToCreated, originalURL)
}

func (s *URLService) BulkCreate(ctx context.Context, urls []model.URLBulkCreateRequest, responseDomain string) ([]model.URLBulkCreateResponse, error) {
	isURLListUnique := false
	var urlList []string

	// проверяем уникальность сгенерированного списка shortURL одним запросом
	for !isURLListUnique {
		generatedURLList, err := utility.BulkGenerateShortURL(len(urls), 10)
		isURLListUnique, err = s.repo.IsURLListUnique(ctx, generatedURLList)
		if err != nil {
			isURLListUnique = false
		}
		isURLListUnique = true
		urlList = generatedURLList
	}

	var toCreateList []model.URLBulkCreate
	for i, url := range urls {
		entity := model.URLBulkCreate{
			CorrelationID: url.CorrelationID,
			ShortURL:      urlList[i],
			OriginalURL:   url.OriginalURL,
		}
		toCreateList = append(toCreateList, entity)
	}

	entities, err := s.repo.BulkCreate(ctx, toCreateList)
	if err != nil {
		return nil, err
	}

	if s.app.Cfg.FileStoragePath != "" {
		fileWriter, err := repository.NewWriter(s.app.Cfg.FileStoragePath)
		if err != nil {
			s.log.Error("could not create file writer",
				zap.String("path", s.app.Cfg.FileStoragePath),
				zap.String("error message", err.Error()),
			)
			return []model.URLBulkCreateResponse{}, err
		}
		defer fileWriter.Close()
		err = fileWriter.BulkWriteEntities(entities)
		if err != nil {
			return []model.URLBulkCreateResponse{}, err
		}
	}

	var response []model.URLBulkCreateResponse
	for _, entity := range entities {
		response = append(response, model.URLBulkCreateResponse{
			CorrelationID: entity.CorrelationID,
			ShortURL:      responseDomain + "/" + entity.ShortURL,
		})
	}

	return response, nil
}

func NewURLService(repo repository.URLRepositoryInterface, app *config.App) *URLService {
	return &URLService{
		repo: repo, app: app,
	}
}

package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
	urlrepo "github.com/artni96/url-shortener/internal/repository/urls"
	"github.com/artni96/url-shortener/internal/utility"
	"go.uber.org/zap"
)

var ErrFailedToCreated = errors.New("could not create ShortURL")

type URLServiceInterface interface {
	GetList(ctx context.Context, responseDomain string) ([]model.URLListEntity, error)
	GetUserList(ctx context.Context, responseDomain string, createdBy int) ([]model.URLListEntity, error)
	GetByShortURL(ctx context.Context, urlID string) (string, error)
	Create(ctx context.Context, entity model.URLCreateRequest, responseDomain string) (string, error)
	BulkCreate(ctx context.Context, urls []model.URLBulkCreateRequest, responseDomain string) ([]model.URLBulkCreateResponse, error)
	Update(ctx context.Context, entity model.URLEntity) (string, error)
	Delete(ctx context.Context, urlID string) error
}
type URLService struct {
	dbRepository       urlrepo.DBURLRepositoryInterface
	inMemoryRepository urlrepo.InMemoryURLRepositoryInterface
	app                *config.App
}

func (s *URLService) GetList(ctx context.Context, responseDomain string) ([]model.URLListEntity, error) {

	var entities []model.URLEntity
	var err error

	if s.dbRepository != nil {
		entities, err = s.dbRepository.GetList(ctx)
	} else {
		entities, err = s.inMemoryRepository.GetList()
	}
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	var response []model.URLListEntity
	for _, data := range entities {
		response = append(response, model.URLListEntity{
			OriginalURL: data.OriginalURL,
			ShortURL:    fmt.Sprintf("%s/%s", responseDomain, data.ShortURL),
		})

	}

	return response, nil
}

func (s *URLService) GetUserList(ctx context.Context, responseDomain string, createdBy int) ([]model.URLListEntity, error) {

	var entities []model.URLEntity
	var err error

	if s.dbRepository != nil {
		entities, err = s.dbRepository.GetUserList(ctx, createdBy)
	} else {
		entities, err = s.inMemoryRepository.GetUserList(createdBy)
	}
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	var response []model.URLListEntity
	for _, data := range entities {
		response = append(response, model.URLListEntity{
			OriginalURL: data.OriginalURL,
			ShortURL:    fmt.Sprintf("%s/%s", responseDomain, data.ShortURL),
		})

	}

	return response, nil
}

func (s *URLService) GetByShortURL(ctx context.Context, shortURL string) (string, error) {
	var originalURL string
	var err error

	if s.dbRepository != nil {
		originalURL, err = s.dbRepository.GetByShortURL(ctx, shortURL)
	} else {
		originalURL, err = s.inMemoryRepository.GetByShortURL(shortURL)
	}

	if err != nil {
		return "", fmt.Errorf("%w", err)
	}

	return originalURL, nil
}

func (s *URLService) Create(ctx context.Context, requestEntity model.URLCreateRequest, responseDomain string) (string, error) {

	for i := range 5 {
		var responseEntity model.URLEntity
		var err error
		shortURL, err := utility.GenerateShortURL(10)
		if err != nil {
			s.app.Logger.Info(
				"urlID creation",
				zap.Int("attempt №", i),
			)
			continue
		}
		inputEntity := model.URLCreate{
			ShortURL:    shortURL,
			OriginalURL: requestEntity.OriginalURL,
			CreatedBy:   requestEntity.CreatedBy,
		}
		if s.dbRepository != nil {

			responseEntity, err = s.dbRepository.Create(ctx, inputEntity)
		} else {
			responseEntity, err = s.inMemoryRepository.Create(inputEntity)
		}

		if err != nil {
			if errors.Is(err, urlrepo.ErrOriginalURLAlreadyExists) {

				s.app.Logger.Info("original url already exists",
					zap.String("originalURL", requestEntity.OriginalURL),
				)

				return fmt.Sprintf("%s/%s", responseDomain, responseEntity.ShortURL), err
			}
			if errors.Is(err, urlrepo.ErrShortURLAlreadyExists) {
				s.app.Logger.Info(
					"shortURL creation",
					zap.Int("attempt №", i),
				)
				continue
			} else {
				s.app.Logger.Error(
					"could not manage to create a short url for ",
					zap.String("url", requestEntity.OriginalURL),
				)
				return "", fmt.Errorf("%w", err)
			}
		}

		if s.dbRepository == nil && s.app.Cfg.FileStoragePath != "" {
			fileWriter, err := urlrepo.NewWriter(s.app.Cfg.FileStoragePath)
			if err != nil {
				s.app.Logger.Error("could not create file writer",
					zap.String("path", s.app.Cfg.FileStoragePath),
					zap.String("error message", err.Error()),
				)
				return "", err
			}
			defer fileWriter.Close()

			err = fileWriter.WriteEntity(&responseEntity)
			if err != nil {
				return "", fmt.Errorf("%w", err)
			}
		}

		return fmt.Sprintf("%s/%s", responseDomain, responseEntity.ShortURL), nil
	}
	return "", fmt.Errorf("%w %s", ErrFailedToCreated, requestEntity.OriginalURL)
}

func (s *URLService) BulkCreate(ctx context.Context, urls []model.URLBulkCreateRequest, responseDomain string) ([]model.URLBulkCreateResponse, error) {
	isURLListUnique := false
	var urlList []string

	// проверяем уникальность сгенерированного списка shortURL одним запросом
	for !isURLListUnique {
		generatedURLList, err := utility.BulkGenerateShortURL(len(urls), 10)
		if s.dbRepository != nil {
			isURLListUnique, err = s.dbRepository.IsURLListUnique(ctx, generatedURLList)
		} else {
			isURLListUnique, err = s.inMemoryRepository.IsURLListUnique(generatedURLList)
		}

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
			CreatedBy:     url.CreatedBy,
		}
		toCreateList = append(toCreateList, entity)
	}

	var entities []model.URLBulkCreate
	var err error
	if s.dbRepository != nil {
		entities, err = s.dbRepository.BulkCreate(ctx, toCreateList)
	} else {
		entities, err = s.inMemoryRepository.BulkCreate(toCreateList)
	}

	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	if s.dbRepository == nil && s.app.Cfg.FileStoragePath != "" {
		fileWriter, err := urlrepo.NewWriter(s.app.Cfg.FileStoragePath)
		if err != nil {
			s.app.Logger.Error("could not create file writer",
				zap.String("path", s.app.Cfg.FileStoragePath),
				zap.String("error message", err.Error()),
			)
			return []model.URLBulkCreateResponse{}, fmt.Errorf("%w", err)
		}
		defer fileWriter.Close()

		entitiesForFile := []model.URLEntity{}

		for _, entity := range entities {
			entitiesForFile = append(entitiesForFile, model.URLEntity{
				OriginalURL: entity.OriginalURL,
				ShortURL:    entity.ShortURL,
			})
		}
		err = fileWriter.BulkWriteEntities(entitiesForFile, false)
		if err != nil {
			return []model.URLBulkCreateResponse{}, fmt.Errorf("%w", err)
		}
	}

	var response []model.URLBulkCreateResponse
	for _, entity := range entities {
		response = append(response, model.URLBulkCreateResponse{
			CorrelationID: entity.CorrelationID,
			ShortURL:      fmt.Sprintf("%s/%s", responseDomain, entity.ShortURL),
		})
	}

	return response, nil
}

func (s *URLService) Update(ctx context.Context, entity model.URLEntity) (string, error) {
	var updatedEntity model.URLEntity
	var err error

	if s.dbRepository != nil {
		updatedEntity, err = s.dbRepository.Update(ctx, entity)
	} else {
		updatedEntity, err = s.inMemoryRepository.Update(entity)
	}
	if err != nil {
		return "", fmt.Errorf("%w", err)
	}
	if s.dbRepository == nil && s.app.Cfg.FileStoragePath != "" {
		entitiesForFile, err := s.inMemoryRepository.GetList()
		fileWriter, err := urlrepo.NewWriter(s.app.Cfg.FileStoragePath)
		if err != nil {
			s.app.Logger.Error("could not create file writer",
				zap.String("path", s.app.Cfg.FileStoragePath),
				zap.String("error message", err.Error()),
			)
			return "", fmt.Errorf("%w", err)
		}
		defer fileWriter.Close()

		err = fileWriter.BulkWriteEntities(entitiesForFile, true)
		if err != nil {
			return "", fmt.Errorf("%w", err)
		}
	}
	return updatedEntity.OriginalURL, nil
}

func (s *URLService) Delete(ctx context.Context, shortURL string) error {
	var err error
	if s.dbRepository != nil {
		err = s.dbRepository.Delete(ctx, shortURL)
	} else {
		err = s.inMemoryRepository.Delete(shortURL)
	}

	if err != nil {
		return fmt.Errorf("%w", err)
	}

	if s.dbRepository == nil && s.app.Cfg.FileStoragePath != "" {
		entitiesForFile, err := s.inMemoryRepository.GetList()
		fileWriter, err := urlrepo.NewWriter(s.app.Cfg.FileStoragePath)
		if err != nil {
			s.app.Logger.Error("could not create file writer",
				zap.String("path", s.app.Cfg.FileStoragePath),
				zap.String("error message", err.Error()),
			)
			return fmt.Errorf("%w", err)
		}
		defer fileWriter.Close()
		err = fileWriter.BulkWriteEntities(entitiesForFile, true)
		if err != nil {
			return fmt.Errorf("%w", err)
		}
	}

	return nil
}

func NewURLService(
	dbRepository urlrepo.DBURLRepositoryInterface,
	inMemoryRepository urlrepo.InMemoryURLRepositoryInterface,
	app *config.App,
) *URLService {
	return &URLService{
		dbRepository: dbRepository, inMemoryRepository: inMemoryRepository, app: app,
	}
}

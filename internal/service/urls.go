package service

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
	urlrepo "github.com/artni96/url-shortener/internal/repository/urls"
	usersrepo "github.com/artni96/url-shortener/internal/repository/users"
	"github.com/artni96/url-shortener/internal/utility"
)

var ErrFailedToCreated = errors.New("could not create ShortURL")
var ErrURLListNotUnique = errors.New("url list is not unique")

// URLServiceInterface encapsulates usecase logic for URL entities.
type URLServiceInterface interface {
	// GetList returns the list of all URL entities.
	GetList(ctx context.Context, responseDomain string) ([]model.URLListEntity, error)
	// GetUserList returns the list of user's URL entities.
	GetUserList(ctx context.Context, responseDomain string, createdBy int) ([]model.URLListEntity, error)
	// GetByShortURL returns a URL entity with original URL value by its short URL.
	GetByShortURL(ctx context.Context, urlID string) (model.GetByShortURLResponse, error)
	// Create saves a new URL entity.
	Create(ctx context.Context, entity model.URLCreateRequest, responseDomain string) (string, error)
	// BulkCreate saves multiple URL entities by one commit.
	BulkCreate(ctx context.Context, urls []model.URLBulkCreateRequest, responseDomain string) ([]model.URLBulkCreateResponse, error)
	// Update saves the changes of a URL entity.
	Update(ctx context.Context, entity model.URLEntity) (string, error)
	// Delete removes a URL entity by its short URL value.
	Delete(ctx context.Context, urlID string) error
	// BulkDelete removes several URL entities by one commit.
	BulkDelete(ctx context.Context, urls []model.URLDelete) error
}

// URLService implements the interaction with URL data through DB or/else a file.
type URLService struct {
	dbRepository       urlrepo.DBURLRepositoryInterface
	inMemoryRepository urlrepo.InMemoryURLRepositoryInterface
	app                *config.App
}

// GetList returns the list of all URL entities.
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

// GetUserList returns the list of user's URL entities.
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

// GetByShortURL returns a URL entity with original URL value by its short URL.
func (s *URLService) GetByShortURL(ctx context.Context, shortURL string) (model.GetByShortURLResponse, error) {
	var entity model.GetByShortURLResponse
	var err error

	if s.dbRepository != nil {
		entity, err = s.dbRepository.GetByShortURL(ctx, shortURL)
	} else {
		entity, err = s.inMemoryRepository.GetByShortURL(shortURL)
	}

	if err != nil {
		return model.GetByShortURLResponse{}, fmt.Errorf("%w", err)
	}

	return entity, nil
}

// Create saves a new URL entity.
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

// BulkCreate saves multiple URL entities by one commit.
func (s *URLService) BulkCreate(ctx context.Context, urls []model.URLBulkCreateRequest, responseDomain string) ([]model.URLBulkCreateResponse, error) {
	isURLListUnique := false
	var urlList []string

	// проверяем уникальность сгенерированного списка shortURL одним запросом
	for !isURLListUnique {
		generatedURLList, err := utility.BulkGenerateShortURL(len(urls), 10)
		if err != nil {
			return nil, fmt.Errorf("%w", err)
		}
		if s.dbRepository != nil {
			isURLListUnique, err = s.dbRepository.IsURLListUnique(ctx, generatedURLList)
			if err != nil {
				return nil, fmt.Errorf("%w", err)
			}
		} else {
			isURLListUnique, err = s.inMemoryRepository.IsURLListUnique(generatedURLList)
			if err != nil {
				return nil, fmt.Errorf("%w", err)
			}
		}

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
				CreatedBy:   entity.CreatedBy,
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

// Update saves the changes of a URL entity.
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
	err = BulkFileUpdate(s)
	if err != nil {
		return "", err
	}
	return updatedEntity.OriginalURL, nil
}

// Delete removes a URL entity by its short URL value.
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

	err = BulkFileUpdate(s)
	if err != nil {
		return err
	}

	return nil
}

// BulkDelete removes several URL entities by one commit.
func (s *URLService) BulkDelete(ctx context.Context, urls []model.URLDelete) error {
	set := make(map[string]struct{})
	var uniqueURLs []model.URLDelete
	var err error
	var errs []error

	for _, url := range urls {
		if _, ok := set[url.ShortURL]; ok {
			continue
		} else {
			set[url.ShortURL] = struct{}{}
			uniqueURLs = append(uniqueURLs, url)
		}
	}

	if s.dbRepository != nil {
		errs, err = s.dbRepository.BulkDelete(ctx, urls)
	} else {
		errs, err = s.inMemoryRepository.BulkDelete(uniqueURLs)
	}

	if err != nil {
		s.app.Logger.Error("failed to bulk delete urls")
		return err
	}
	for _, e := range errs {
		s.app.Logger.Info("failed to delete url", zap.Error(e))
	}

	err = BulkFileUpdate(s)
	if err != nil {
		return err
	}
	return nil
}

// BulkFileUpdate updates the file with new data (users and urls).
func BulkFileUpdate(s *URLService) error {
	filepath := s.app.Cfg.FileStoragePath

	if s.dbRepository == nil && filepath != "" {
		fileReader, err := usersrepo.NewFileScanner(filepath)
		if fileReader == nil {
			return nil
		}
		defer func(fileReader *usersrepo.FileScanner) {
			err = fileReader.Close()
			if err != nil {
				s.app.Logger.Info("could not close file reader", zap.String("filepath", filepath))
			}
		}(fileReader)

		if err != nil {
			s.app.Logger.Info("could not initialize NewFileScanner",
				zap.String("error", err.Error()))
			return nil
		}
		usersForFile, err := fileReader.CollectData()
		if err != nil {
			s.app.Logger.Info("could not collect data from file",
				zap.String("filepath", filepath),
				zap.String("error", err.Error()))
			return fmt.Errorf("could not collect data from file: %w", err)
		}

		urlsForFile, err := s.inMemoryRepository.GetList()
		if err != nil {
			return fmt.Errorf("failed to get url list: %w", err)
		}
		urlsFileWriter, err := urlrepo.NewWriter(filepath)
		if err != nil {
			s.app.Logger.Error("could not create file writer",
				zap.String("path", filepath),
				zap.String("error message", err.Error()),
			)
			return fmt.Errorf("%w", err)
		}
		defer urlsFileWriter.Close()
		err = urlsFileWriter.BulkWriteEntities(urlsForFile, true)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		usersFileWriter, err := usersrepo.NewWriter(filepath)
		if err != nil {
			s.app.Logger.Error("could not create file writer",
				zap.String("path", filepath),
				zap.String("error message", err.Error()),
			)
			return fmt.Errorf("%w", err)
		}
		defer usersFileWriter.Close()
		err = usersFileWriter.BulkWriteEntities(usersForFile, false)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		return nil
	}
	return nil
}

// NewURLService initializes a new URLService.
func NewURLService(
	dbRepository urlrepo.DBURLRepositoryInterface,
	inMemoryRepository urlrepo.InMemoryURLRepositoryInterface,
	app *config.App,
) *URLService {
	return &URLService{
		dbRepository: dbRepository, inMemoryRepository: inMemoryRepository, app: app,
	}
}

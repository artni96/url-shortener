package repository

import (
	"errors"
	"fmt"
	"sync"

	"github.com/artni96/url-shortener/internal/model"
)

var ErrURLIDDuplicate = errors.New("найден в БД")
var ErrURLNotFound = errors.New("url not found")

type URLRepositoryInterface interface {
	GetByID(urlID string) (string, error)
	Create(urlStr, urlID string) (model.URLEntity, error)
}
type LocalURLRepository struct {
	mu sync.RWMutex
	//urls map[string]string
	urls []model.URLEntity
}

func (repo *LocalURLRepository) Create(urlStr, urlID string) (model.URLEntity, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	_, err := repo.GetByIDUnlocked(urlID)
	newEntity := model.URLEntity{}
	if err != nil {
		if !errors.Is(err, ErrURLNotFound) {
			return newEntity, err
		}

	}
	curID := getLastID(repo.urls)
	newEntity.ID = curID
	newEntity.OriginalURL = urlStr
	newEntity.ShortURL = urlID
	repo.urls = append(repo.urls, newEntity)
	return newEntity, nil
}

func (repo *LocalURLRepository) GetByID(urlID string) (string, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	return repo.GetByIDUnlocked(urlID)
}

func (repo *LocalURLRepository) GetByIDUnlocked(urlID string) (string, error) {
	if len(repo.urls) == 0 {
		return "", ErrURLNotFound
	}

	for _, entity := range repo.urls {
		if entity.ShortURL == urlID {
			return entity.OriginalURL, nil
		}
	}
	return "", fmt.Errorf("%s %w", urlID, ErrURLNotFound)
}

func NewURLRepository() *LocalURLRepository {
	return &LocalURLRepository{urls: []model.URLEntity{}}
}

func getLastID(storage []model.URLEntity) int {
	lastID := 0
	for _, lineVal := range storage {
		if lineVal.ID > lastID {
			lastID = lineVal.ID
		}
	}
	return lastID
}

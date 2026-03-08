package repository

import (
	"errors"
	"sync"

	"github.com/artni96/url-shortener/internal/utility"
)

type URLRepositoryInterface interface {
	Get(urlID string) (string, error)
	Create(urlStr string) (string, error)
}
type LocalURLRepository struct {
	mu   sync.RWMutex
	urls map[string]string
}

func (repo *LocalURLRepository) Create(urlStr string) (string, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	urlID, err := utility.GenerateID(10)
	if err != nil {
		return "", err
	}
	_, ok := repo.urls[urlID]
	for ok {
		urlID, err = utility.GenerateID(10)
		if err != nil {
			return "", err
		}
		_, ok = repo.urls[urlID]
	}
	repo.urls[urlID] = urlStr
	return urlID, nil
}

func (repo *LocalURLRepository) Get(urlID string) (string, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	urlStr, ok := repo.urls[urlID]
	if !ok {
		return "", errors.New("URL не найден")
	}
	return urlStr, nil
}

func NewURLRepository() *LocalURLRepository {
	return &LocalURLRepository{urls: make(map[string]string)}
}

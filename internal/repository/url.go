package repository

import (
	"errors"
	"sync"
)

//var localDB = map[string]string{}

type LocalURLRepository struct {
	mu   sync.RWMutex
	urls map[string]string
}

func (repo *LocalURLRepository) Create(urlStr string, urlID string) (string, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
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

type URLRepositoryInterface interface {
	Get(urlID string) (string, error)
	Create(urlStr, urlID string) (string, error)
}

func NewURLRepository() *LocalURLRepository {
	return &LocalURLRepository{urls: make(map[string]string)}
}

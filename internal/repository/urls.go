package repository

import (
	"errors"
	"fmt"
	"sync"
)

var ErrURLIDDuplicate = errors.New("найден в БД")

type URLRepositoryInterface interface {
	Get(urlID string) (string, error)
	Create(urlStr, urlID string) (string, error)
}
type LocalURLRepository struct {
	mu   sync.RWMutex
	urls map[string]string
}

func (repo *LocalURLRepository) Create(urlStr, urlID string) (string, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	_, ok := repo.urls[urlID]
	if ok {
		return "", errors.New("urlID найден в БД")
	}
	repo.urls[urlID] = urlStr
	return urlID, nil
}

func (repo *LocalURLRepository) Get(urlID string) (string, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	urlStr, ok := repo.urls[urlID]
	if !ok {
		return "", fmt.Errorf("%s %w", urlID, ErrURLIDDuplicate)
	}
	return urlStr, nil
}

func NewURLRepository() *LocalURLRepository {
	return &LocalURLRepository{urls: make(map[string]string)}
}

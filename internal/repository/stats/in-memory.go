package stats

import (
	"sync"

	"go.uber.org/zap"
)

type InMemoryStatsRepository struct {
	mu     sync.Mutex
	logger *zap.Logger
}

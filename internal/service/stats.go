package service

import (
	"context"
	"fmt"

	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/repository/stats"
	"github.com/jmoiron/sqlx"
)

type StatsServiceInterface interface {
	Get(ctx context.Context) (model.StatsResponse, error)
}

type StatsService struct {
	dbRepository stats.DBStatsRepositoryInterface
	db           *sqlx.DB
	userService  *UserService
	urlService   *URLService
}

func (s *StatsService) Get(ctx context.Context) (model.StatsResponse, error) {
	var response model.StatsResponse
	if s.db != nil {
		response, err := s.dbRepository.Get(ctx)
		if err != nil {
			return model.StatsResponse{}, fmt.Errorf("%w", err)
		}
		return response, nil
	}
	response.Users = s.userService.GetStats()
	response.URLs = s.urlService.GetStats()
	return response, nil
}

func NewStatsService(dbRepository stats.DBStatsRepositoryInterface, db *sqlx.DB, userService *UserService, urlService *URLService) *StatsService {
	return &StatsService{dbRepository: dbRepository, db: db, userService: userService, urlService: urlService}
}

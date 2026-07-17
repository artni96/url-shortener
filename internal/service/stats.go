package service

import (
	"context"
	"fmt"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/repository/stats"
)

type StatsServiceInterface interface {
	Get(ctx context.Context) (model.StatsResponse, error)
}

type StatsService struct {
	dbRepository stats.DBStatsRepositoryInterface
	app          *config.App
	userService  *UserService
	urlService   *URLService
}

func (s *StatsService) Get(ctx context.Context) (model.StatsResponse, error) {
	var response model.StatsResponse
	if s.app.DB != nil {
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

func NewStatsService(dbRepository stats.DBStatsRepositoryInterface, app *config.App, userService *UserService, urlService *URLService) *StatsService {
	return &StatsService{dbRepository: dbRepository, app: app, userService: userService, urlService: urlService}
}

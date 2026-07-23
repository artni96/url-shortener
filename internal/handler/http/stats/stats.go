package stats

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/artni96/url-shortener/internal/config"
	middlewares2 "github.com/artni96/url-shortener/internal/handler/http/middlewares"
	logger2 "github.com/artni96/url-shortener/internal/logger"
	"github.com/artni96/url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type StatsHandler struct {
	logger  *zap.Logger
	service service.StatsServiceInterface
}

func NewStatsHandler(logger *zap.Logger, service service.StatsServiceInterface) *StatsHandler {
	return &StatsHandler{logger: logger, service: service}
}

func (s *StatsHandler) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	stats, err := s.service.Get(r.Context())
	if err != nil {
		s.logger.Error("failed to get stats", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response, err := json.Marshal(stats)
	if err != nil {
		s.logger.Error("failed to marshal", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func StatsRouter(logger *zap.Logger, service service.StatsServiceInterface, trustedSubnet string) chi.Router {
	r := chi.NewRouter()

	r.Use(middlewares2.PanicRecoverer(logger))
	r.Use(middlewares2.CheckIP(logger, trustedSubnet))
	r.Use(middleware.RealIP)
	r.Use(logger2.RequestLoggerMiddleware(logger))
	r.Use(config.GzipMiddleware)

	urlHandler := NewStatsHandler(logger, service)

	r.Route("/", func(r chi.Router) {
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Method %s is forbidden", r.Method)))
		})
		r.Get("/stats", urlHandler.GetStatsHandler)
	})
	return r
}

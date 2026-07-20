package stats

import (
	"context"
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
	ctx     *context.Context
	service service.StatsServiceInterface
}

func NewStatsHandler(ctx context.Context, logger *zap.Logger, service service.StatsServiceInterface) *StatsHandler {
	return &StatsHandler{logger: logger, ctx: &ctx, service: service}
}

func (s *StatsHandler) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := s.service.Get(*s.ctx)
	if err != nil {
		s.logger.Error("failed to get stats", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
	}
	response, err := json.Marshal(stats)
	if err != nil {
		s.logger.Error("failed to marshal", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func StatsRouter(ctx context.Context, app *config.App, service service.StatsServiceInterface, trustedSubnet string) chi.Router {
	r := chi.NewRouter()

	r.Use(middlewares2.PanicRecoverer(app.Logger))
	r.Use(middlewares2.CheckIP(app.Logger, trustedSubnet))
	r.Use(middleware.RealIP)
	r.Use(logger2.RequestLoggerMiddleware(app.Logger))
	r.Use(config.GzipMiddleware)

	urlHandler := NewStatsHandler(ctx, app.Logger, service)

	r.Route("/", func(r chi.Router) {
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Method %s is forbidden", r.Method)))
		})
		r.Get("/stats", urlHandler.GetStatsHandler)
	})
	return r
}

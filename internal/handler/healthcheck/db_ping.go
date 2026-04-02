package healthcheck

import (
	"context"
	"errors"
	"net/http"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type Handler struct {
	db  *sqlx.DB
	log *zap.Logger
	ctx *context.Context
}

func NewHealthCheckHandler(ctx *context.Context, app *config.App) *Handler {
	return &Handler{
		db:  app.DB,
		log: app.Logger,
		ctx: ctx,
	}
}

func (h *Handler) PingHandler(w http.ResponseWriter, r *http.Request) {

	if h.db == nil {
		h.log.Info("unable to ping database", zap.Error(errors.New("unable to ping database")))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := h.db.PingContext(*h.ctx); err != nil {
		h.log.Info("unable to ping database", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	h.log.Info("successful pinging database")
	return
}

func HealthCheckRouter(ctx *context.Context, app *config.App) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(logger.RequestLoggerMiddleware(app.Logger))
	r.Use(middleware.Recoverer)
	r.Use(config.GzipMiddleware)

	handler := NewHealthCheckHandler(ctx, app)
	r.Get("/", handler.PingHandler)
	return r
}

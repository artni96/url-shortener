package healthcheck

import (
	"context"
	"net/http"

	"github.com/artni96/url-shortener/internal/handler/http/middlewares"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/artni96/url-shortener/internal/config"
	appLogger "github.com/artni96/url-shortener/internal/logger"
)

type Handler struct {
	db  *sqlx.DB
	log *zap.Logger
	ctx *context.Context
}

func NewHealthCheckHandler(ctx *context.Context, db *sqlx.DB, logger *zap.Logger) *Handler {
	return &Handler{
		db:  db,
		log: logger,
		ctx: ctx,
	}
}

// PingHandler godoc
// @Summary Checks DB connection
// @Description Checks DB connection
// @Tags health check
// @Accept json
// @Produce json
// @Success 200
// @Failure 500
// @Router /ping [get]
func (h *Handler) PingHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(*h.ctx); err != nil {
		h.log.Info("unable to ping database", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	h.log.Info("successful pinging database")
	return
}

func HealthCheckRouter(ctx *context.Context, logger *zap.Logger, db *sqlx.DB) http.Handler {
	r := chi.NewRouter()

	r.Use(middlewares.PanicRecoverer(logger))
	r.Use(middleware.RealIP)
	r.Use(appLogger.RequestLoggerMiddleware(logger))
	r.Use(config.GzipMiddleware)

	handler := NewHealthCheckHandler(ctx, db, logger)
	r.Get("/", handler.PingHandler)
	return r
}

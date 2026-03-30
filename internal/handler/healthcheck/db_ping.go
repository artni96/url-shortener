package healthcheck

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

type Handler struct {
	db  *sql.DB
	log *zap.Logger
}

func NewHealthCheckHandler(db *sql.DB, log *zap.Logger) *Handler {
	return &Handler{
		db:  db,
		log: log,
	}
}

func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		h.log.Error("unable to ping database", zap.Error(err))
		return
	}

	w.WriteHeader(http.StatusOK)
	h.log.Info("successful pinging database")
	return
}

func HealthCheckRouter(db *sql.DB, log *zap.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(logger.RequestLoggerMiddleware(log))
	r.Use(middleware.Recoverer)
	r.Use(config.GzipMiddleware)

	handler := NewHealthCheckHandler(db, log)
	r.Get("/", handler.Ping)
	return r
}

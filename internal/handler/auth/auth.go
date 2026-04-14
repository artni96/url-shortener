package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/handler"
	"github.com/artni96/url-shortener/internal/handler/middlewares"
	"github.com/artni96/url-shortener/internal/logger"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type AuthHandler struct {
	service service.AuthService
	logger  *zap.Logger
	ctx     *context.Context
}

func NewAuthHandler(ctx *context.Context, service service.AuthService, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		ctx:     ctx,
		service: service,
		logger:  logger,
	}
}

func (h *AuthHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body model.UserCreateRequest
	var buf bytes.Buffer

	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		handler.ErrorResponse(w, err.Error(), http.StatusBadRequest, h.logger)
		return
	}

	if err = json.Unmarshal(buf.Bytes(), &body); err != nil {
		handler.ErrorResponse(w, err.Error(), http.StatusBadRequest, h.logger)
		return
	}

	err = h.service.Create(r.Context(), body)
	if err != nil {
		handler.ErrorResponse(w, err.Error(), http.StatusBadRequest, h.logger)
		return
	}
	w.WriteHeader(http.StatusCreated)
	return
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body model.UserLogin
	var buf bytes.Buffer

	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		handler.ErrorResponse(w, err.Error(), http.StatusBadRequest, h.logger)
		return
	}

	if err = json.Unmarshal(buf.Bytes(), &body); err != nil {
		handler.ErrorResponse(w, err.Error(), http.StatusBadRequest, h.logger)
		return
	}

	token, err := h.service.Login(r.Context(), body)
	if err != nil {
		handler.ErrorResponse(w, err.Error(), http.StatusBadRequest, h.logger)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "Authorization",
		Value:    fmt.Sprintf("Bearer %s", token),
		Expires:  time.Now().Add(service.TOKEN_EXP),
		HttpOnly: true,
	})
	w.WriteHeader(http.StatusCreated)
	return
}

func AuthRouter(ctx *context.Context, app *config.App, service service.AuthService) chi.Router {
	r := chi.NewRouter()

	r.Use(middlewares.PanicRecoverer(app.Logger))
	r.Use(middleware.RealIP)
	r.Use(logger.RequestLoggerMiddleware(app.Logger))
	r.Use(config.GzipMiddleware)

	h := NewAuthHandler(ctx, service, app.Logger)

	r.Route("/", func(r chi.Router) {
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Method %s is forbidden", r.Method)))
		})
		r.Post("/signup", h.Create)
		r.Post("/login", h.Login)
	})
	return r
}

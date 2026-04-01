package urls

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/logger"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

const urlPattern = `^https?:\/\/`

type URLHandler struct {
	responseURL string
	urlService  service.URLServiceInterface
	appLogger   *zap.Logger
	ctx         *context.Context
}

func NewURLHandler(ctx *context.Context, cfg *config.Config, urlService service.URLServiceInterface, appLogger *zap.Logger) *URLHandler {
	return &URLHandler{
		responseURL: cfg.ResponseURL,
		urlService:  urlService,
		appLogger:   appLogger,
		ctx:         ctx,
	}
}

func (h *URLHandler) ShortenURLHandler(w http.ResponseWriter, r *http.Request) {
	var responseData model.URLCreateResponse

	var body model.URLCreateRequest
	var buf bytes.Buffer

	_, err := buf.ReadFrom(r.Body)
	defer r.Body.Close()

	if err != nil {
		errMessage := "invalid request - empty body"
		ErrorResponse(w, errMessage, http.StatusBadRequest, h.appLogger)
		return
	}

	if err = json.Unmarshal(buf.Bytes(), &body); err != nil {
		errMessage := "could not unmarshal request body"
		ErrorResponse(w, errMessage, http.StatusBadRequest, h.appLogger)
		return
	}

	if !isURLCorrect(body.URL) {
		errMessage := fmt.Sprintf("url '%s' is invalid", body.URL)
		ErrorResponse(w, errMessage, http.StatusBadRequest, h.appLogger)
		return
	}

	urlID, err := h.urlService.Create(*h.ctx, body.URL)
	if err != nil {
		if errors.Is(err, service.ErrFailedToCreated) {
			errMessage := err.Error()
			ErrorResponse(w, errMessage, http.StatusInternalServerError, h.appLogger)
			return
		} else {
			errMessage := fmt.Sprintf("Could not create short URL for %s", body.URL)
			ErrorResponse(w, errMessage, http.StatusInternalServerError, h.appLogger)
			return
		}
	}

	responseData.Result = fmt.Sprintf("%s/%s", h.responseURL, urlID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	resp, err := json.Marshal(responseData)
	if err != nil {
		errMessage := fmt.Sprintf("Could not create short URL for %s", body.URL)
		ErrorResponse(w, errMessage, http.StatusInternalServerError, h.appLogger)
		return
	}
	w.Write(resp)
}

func (h *URLHandler) CreateURLHandler(w http.ResponseWriter, r *http.Request) {

	originalURL, err := io.ReadAll(r.Body)
	if err != nil && err.Error() != "EOF" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Could not read request body"))
		return
	}

	urlStr := string(originalURL)
	if urlStr == "" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Empty request body"))
		return
	}

	if !isURLCorrect(urlStr) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Incorrect request body"))
		return
	}

	urlID, err := h.urlService.Create(*h.ctx, urlStr)
	if err != nil {
		if errors.Is(err, service.ErrFailedToCreated) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		} else {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Could not handle the request"))
		}
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("%s/%s", h.responseURL, urlID)))

	defer r.Body.Close()
}

func (h *URLHandler) GetURLHandler(w http.ResponseWriter, r *http.Request) {
	redirectTo, err := h.urlService.GetByShortURL(*h.ctx, strings.TrimPrefix(r.URL.Path, "/"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("URL not found"))
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Location", redirectTo)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func URLRouter(ctx *context.Context, cfg *config.Config, urlService service.URLServiceInterface, appLogger *zap.Logger) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(logger.RequestLoggerMiddleware(appLogger))
	r.Use(middleware.Recoverer)
	r.Use(config.GzipMiddleware)

	urlHandler := NewURLHandler(ctx, cfg, urlService, appLogger)

	r.Route("/", func(r chi.Router) {
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Method %s is forbidden", r.Method)))
		})
		r.Post("/", urlHandler.CreateURLHandler)
		r.Post("/api/shorten", urlHandler.ShortenURLHandler)

		r.Get("/{id}", urlHandler.GetURLHandler)
	})
	return r
}

func isURLCorrect(url string) bool {
	matched, err := regexp.MatchString(urlPattern, url)
	if err != nil {
		return false
	}
	if matched {
		return true
	}
	return false
}

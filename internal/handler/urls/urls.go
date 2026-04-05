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
	"github.com/artni96/url-shortener/internal/repository"
	"github.com/artni96/url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

const urlPattern = `^https?:\/\/`

type URLHandler struct {
	responseDomain string
	service        service.URLServiceInterface
	logger         *zap.Logger
	ctx            *context.Context
}

func NewURLHandler(ctx *context.Context, app *config.App, urlService service.URLServiceInterface) *URLHandler {
	return &URLHandler{
		responseDomain: app.Cfg.ResponseDomain,
		service:        urlService,
		logger:         app.Logger,
		ctx:            ctx,
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
		ErrorResponse(w, errMessage, http.StatusBadRequest, h.logger)
		return
	}

	if err = json.Unmarshal(buf.Bytes(), &body); err != nil {
		errMessage := "could not unmarshal request body"
		ErrorResponse(w, errMessage, http.StatusBadRequest, h.logger)
		return
	}

	if !isURLCorrect(body.URL) {
		errMessage := fmt.Sprintf("url '%s' is invalid", body.URL)
		ErrorResponse(w, errMessage, http.StatusBadRequest, h.logger)
		return
	}

	shortURL, err := h.service.Create(*h.ctx, body.URL, h.responseDomain)
	responseData.Result = shortURL
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		if errors.Is(err, service.ErrFailedToCreated) {
			errMessage := err.Error()
			ErrorResponse(w, errMessage, http.StatusInternalServerError, h.logger)
			return
		} else if errors.Is(err, repository.ErrOriginalURLAlreadyExists) {

			w.WriteHeader(http.StatusConflict)
			resp, err := json.Marshal(responseData)
			if err != nil {
				errMessage := fmt.Sprintf("Could not create short URL for %s", body.URL)
				ErrorResponse(w, errMessage, http.StatusInternalServerError, h.logger)
				return
			}
			w.Write(resp)
			return
		} else {
			errMessage := fmt.Sprintf("Could not create short URL for %s", body.URL)
			ErrorResponse(w, errMessage, http.StatusInternalServerError, h.logger)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	resp, err := json.Marshal(responseData)
	if err != nil {
		errMessage := fmt.Sprintf("Could not create short URL for %s", body.URL)
		ErrorResponse(w, errMessage, http.StatusInternalServerError, h.logger)
		return
	}
	w.Write(resp)
}

func (h *URLHandler) BulkCreateURLHandler(w http.ResponseWriter, r *http.Request) {

	var body []model.URLBulkCreateRequest
	var buf bytes.Buffer

	_, err := buf.ReadFrom(r.Body)
	defer r.Body.Close()

	if err != nil {
		errMessage := "invalid request - empty body"
		ErrorResponse(w, errMessage, http.StatusBadRequest, h.logger)
		return
	}

	if err = json.Unmarshal(buf.Bytes(), &body); err != nil {
		errMessage := "could not unmarshal request body"
		ErrorResponse(w, errMessage, http.StatusBadRequest, h.logger)
		return
	}

	// валидация входных данных
	for _, url := range body {
		if url.OriginalURL == "" || url.CorrelationID == "" {
			errMessage := "invalid body request"
			ErrorResponse(w, errMessage, http.StatusBadRequest, h.logger)
			return
		}
	}

	responseData, err := h.service.BulkCreate(*h.ctx, body, h.responseDomain)
	if err != nil {
		if errors.As(err, &repository.ErrShortURLAlreadyExists) {
			errMessage := err.Error()
			ErrorResponse(w, errMessage, http.StatusBadRequest, h.logger)
			return
		}
		errorMessage := "could not bulk create short URL"
		ErrorResponse(w, errorMessage, http.StatusInternalServerError, h.logger)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	resp, err := json.Marshal(responseData)
	if err != nil {
		errMessage := "could not marshal request body"
		ErrorResponse(w, errMessage, http.StatusInternalServerError, h.logger)
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

	shortURL, err := h.service.Create(*h.ctx, urlStr, h.responseDomain)
	w.Header().Set("Content-Type", "text/plain")
	if err != nil {
		if errors.Is(err, repository.ErrOriginalURLAlreadyExists) {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(shortURL))
			return
		}

		if errors.Is(err, service.ErrFailedToCreated) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		} else {

			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Could not handle the request"))
		}
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))

	defer r.Body.Close()
}

func (h *URLHandler) GetURLHandler(w http.ResponseWriter, r *http.Request) {
	redirectTo, err := h.service.GetByShortURL(*h.ctx, strings.TrimPrefix(r.URL.Path, "/"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("URL not found"))
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Location", redirectTo)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *URLHandler) GetListHandler(w http.ResponseWriter, r *http.Request) {
	urlList, err := h.service.GetList(*h.ctx, h.responseDomain)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Could not get the list of urls"))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp, err := json.Marshal(urlList)
	if err != nil {
		errMessage := "could not marshal request body"
		ErrorResponse(w, errMessage, http.StatusInternalServerError, h.logger)
		return
	}
	w.Write(resp)
}

func (h *URLHandler) DeleteURLHandler(w http.ResponseWriter, r *http.Request) {
	shortURL := strings.TrimPrefix(r.URL.Path, "/")
	err := h.service.Delete(*h.ctx, shortURL)
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		if errors.Is(err, repository.ErrURLNotFound) {
			errMessage := fmt.Sprintf("url with short url '%s' not found", shortURL)
			ErrorResponse(w, errMessage, http.StatusBadRequest, h.logger)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Could not delete the short URL"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func URLRouter(ctx *context.Context, app *config.App, urlService service.URLServiceInterface) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(logger.RequestLoggerMiddleware(app.Logger))
	r.Use(middleware.Recoverer)
	r.Use(config.GzipMiddleware)

	urlHandler := NewURLHandler(ctx, app, urlService)

	r.Route("/", func(r chi.Router) {
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Method %s is forbidden", r.Method)))
		})
		r.Post("/api/shorten", urlHandler.ShortenURLHandler)
		r.Post("/api/shorten/batch", urlHandler.BulkCreateURLHandler)
		r.Get("/", urlHandler.GetListHandler)
		r.Post("/", urlHandler.CreateURLHandler)
		r.Get("/{id}", urlHandler.GetURLHandler)
		r.Delete("/{id}", urlHandler.DeleteURLHandler)

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

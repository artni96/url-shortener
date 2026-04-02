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
	logger      *zap.Logger
	ctx         *context.Context
}

func NewURLHandler(ctx *context.Context, app *config.App, urlService service.URLServiceInterface) *URLHandler {
	return &URLHandler{
		responseURL: app.Cfg.ResponseURL,
		urlService:  urlService,
		logger:      app.Logger,
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

	urlID, err := h.urlService.Create(*h.ctx, body.URL)
	if err != nil {
		if errors.Is(err, service.ErrFailedToCreated) {
			errMessage := err.Error()
			ErrorResponse(w, errMessage, http.StatusInternalServerError, h.logger)
			return
		} else {
			errMessage := fmt.Sprintf("Could not create short URL for %s", body.URL)
			ErrorResponse(w, errMessage, http.StatusInternalServerError, h.logger)
			return
		}
	}

	responseData.Result = fmt.Sprintf("%s/%s", h.responseURL, urlID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	resp, err := json.Marshal(responseData)
	if err != nil {
		errMessage := fmt.Sprintf("Could not create short URL for %s", body.URL)
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

func (h *URLHandler) GetListHandler(w http.ResponseWriter, r *http.Request) {
	urlList, err := h.urlService.GetList(*h.ctx)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Could not get the list of urls"))
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	if err = json.NewEncoder(w).Encode(urlList); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
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
		r.Get("/", urlHandler.GetListHandler)
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

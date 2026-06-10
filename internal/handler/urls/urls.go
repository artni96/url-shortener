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
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/handler"
	"github.com/artni96/url-shortener/internal/handler/middlewares"
	"github.com/artni96/url-shortener/internal/logger"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/repository/urls"
	"github.com/artni96/url-shortener/internal/service"
)

const urlPattern = `^https?:\/\/`

type URLHandler struct {
	responseDomain string
	urlService     service.URLServiceInterface
	userService    service.UserServiceInterface
	logger         *zap.Logger
	ctx            *context.Context
	cfg            *config.Config
	auditChan      chan<- model.AuditEntity
}

func NewURLHandler(ctx *context.Context, app *config.App, urlService service.URLServiceInterface, userService service.UserServiceInterface, cfg *config.Config) *URLHandler {
	return &URLHandler{
		responseDomain: app.Cfg.ResponseDomain,
		urlService:     urlService,
		userService:    userService,
		logger:         app.Logger,
		ctx:            ctx,
		cfg:            cfg,
		auditChan:      app.AuditChan,
	}
}

func (h *URLHandler) ShortenURLHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := getOrCreateUserID(h, w, r)
	if err != nil {
		handler.ErrorResponse(w, "internal server error", http.StatusInternalServerError, h.logger)
		return
	}

	var responseData model.URLCreateResponse
	var body model.URLCreateRequest
	var buf bytes.Buffer

	_, err = buf.ReadFrom(r.Body)
	defer r.Body.Close()

	if err != nil {
		handler.ErrorResponse(w, "invalid request - empty body", http.StatusBadRequest, h.logger)
		return
	}

	if err = json.Unmarshal(buf.Bytes(), &body); err != nil {
		handler.ErrorResponse(w, "could not unmarshal request body", http.StatusBadRequest, h.logger)
		return
	}
	body.CreatedBy = userID

	if !isURLCorrect(body.OriginalURL) {
		handler.ErrorResponse(w, fmt.Sprintf("url '%s' is invalid", body.OriginalURL), http.StatusBadRequest, h.logger)
		return
	}

	shortURL, err := h.urlService.Create(*h.ctx, body, h.responseDomain)

	auditEntity := model.AuditEntity{
		Ts:     time.Now().Unix(),
		URL:    body.OriginalURL,
		UserID: userID,
		Action: "shorten",
	}
	if h.auditChan != nil {
		h.auditChan <- auditEntity
	}

	responseData.Result = shortURL
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		if errors.Is(err, service.ErrFailedToCreated) {
			handler.ErrorResponse(w, err.Error(), http.StatusInternalServerError, h.logger)
			return
		} else if errors.Is(err, urls.ErrOriginalURLAlreadyExists) {
			w.WriteHeader(http.StatusConflict)
			resp, err := json.Marshal(responseData)
			if err != nil {
				handler.ErrorResponse(w, fmt.Sprintf("Could not create short URL for %s", body.OriginalURL), http.StatusInternalServerError, h.logger)
				return
			}
			w.Write(resp)
			return
		}
		handler.ErrorResponse(w, fmt.Sprintf("Could not create short URL for %s", body.OriginalURL), http.StatusInternalServerError, h.logger)
		return

	}

	w.WriteHeader(http.StatusCreated)
	resp, err := json.Marshal(responseData)
	if err != nil {
		handler.ErrorResponse(w, fmt.Sprintf("Could not create short URL for %s", body.OriginalURL), http.StatusInternalServerError, h.logger)
		return
	}
	w.Write(resp)
}

func (h *URLHandler) BulkCreateURLHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := getOrCreateUserID(h, w, r)
	if err != nil {
		handler.ErrorResponse(w, "internal server error", http.StatusInternalServerError, h.logger)
		return
	}

	var body []model.URLBulkCreateRequest
	var buf bytes.Buffer

	_, err = buf.ReadFrom(r.Body)
	defer r.Body.Close()

	if err != nil {
		handler.ErrorResponse(w, "invalid request - empty body", http.StatusBadRequest, h.logger)
		return
	}

	if err = json.Unmarshal(buf.Bytes(), &body); err != nil {
		handler.ErrorResponse(w, "could not unmarshal request body", http.StatusBadRequest, h.logger)
		return
	}

	// валидация входных данных
	for i, url := range body {
		if url.OriginalURL == "" || url.CorrelationID == "" {
			handler.ErrorResponse(w, "invalid body request", http.StatusBadRequest, h.logger)
			return
		}
		body[i].CreatedBy = userID
	}

	responseData, err := h.urlService.BulkCreate(*h.ctx, body, h.responseDomain)
	if err != nil {
		if errors.Is(err, urls.ErrShortURLAlreadyExists) {
			handler.ErrorResponse(w, err.Error(), http.StatusBadRequest, h.logger)
			return
		} else if errors.Is(err, urls.ErrDuplicatedURL) {
			handler.ErrorResponse(w, err.Error(), http.StatusConflict, h.logger)
			return
		}
		handler.ErrorResponse(w, "could not bulk create short URL", http.StatusInternalServerError, h.logger)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	resp, err := json.Marshal(responseData)
	if err != nil {
		handler.ErrorResponse(w, "could not marshal request body", http.StatusInternalServerError, h.logger)
		return
	}
	w.Write(resp)
}

func (h *URLHandler) CreateURLHandler(w http.ResponseWriter, r *http.Request) {
	originalURL, err := io.ReadAll(r.Body)
	defer r.Body.Close()

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
	entity := model.URLCreateRequest{
		OriginalURL: urlStr,
	}
	userID, err := getOrCreateUserID(h, w, r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	entity.CreatedBy = userID
	shortURL, err := h.urlService.Create(*h.ctx, entity, h.responseDomain)

	auditEntity := model.AuditEntity{
		Ts:     time.Now().Unix(),
		URL:    urlStr,
		UserID: userID,
		Action: "shorten",
	}
	if h.auditChan != nil {
		h.auditChan <- auditEntity
	}

	w.Header().Set("Content-Type", "text/plain")
	if err != nil {
		if errors.Is(err, urls.ErrOriginalURLAlreadyExists) {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(shortURL))
			return
		}

		if errors.Is(err, service.ErrFailedToCreated) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		} else {

			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Could not handle the request"))
			return
		}
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

func (h *URLHandler) GetURLHandler(w http.ResponseWriter, r *http.Request) {
	entity, err := h.urlService.GetByShortURL(*h.ctx, strings.TrimPrefix(r.URL.Path, "/"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("URL not found"))
		return
	}
	if entity.IsDeleted == true {
		w.WriteHeader(http.StatusGone)
		return
	}

	auditEntity := model.AuditEntity{
		Ts:     time.Now().Unix(),
		URL:    entity.OriginalURL,
		Action: "shorten",
	}
	if h.auditChan != nil {
		h.auditChan <- auditEntity
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Location", entity.OriginalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *URLHandler) GetListHandler(w http.ResponseWriter, r *http.Request) {
	urlList, err := h.urlService.GetList(*h.ctx, h.responseDomain)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Could not get the list of urls"))
		return
	}
	w.WriteHeader(http.StatusOK)
	resp, err := json.Marshal(urlList)
	if err != nil {
		handler.ErrorResponse(w, "could not marshal request body", http.StatusInternalServerError, h.logger)
		return
	}
	w.Write(resp)
}

func (h *URLHandler) GetUserListHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := getOrCreateUserID(h, w, r)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if userID == -1 {
		w.WriteHeader(http.StatusNoContent)
		return
	} else {
		urlList, err := h.urlService.GetUserList(*h.ctx, h.responseDomain, userID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Could not get the list of urls"))
			return
		}

		if len(urlList) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusOK)
		resp, err := json.Marshal(urlList)
		if err != nil {
			handler.ErrorResponse(w, "could not marshal request body", http.StatusInternalServerError, h.logger)
			return
		}
		w.Write(resp)
	}

}

func (h *URLHandler) UpdateURLHandler(w http.ResponseWriter, r *http.Request) {
	shortURL := strings.TrimPrefix(r.URL.Path, "/")

	var body model.URLCreateRequest
	var buf bytes.Buffer

	_, err := buf.ReadFrom(r.Body)
	defer r.Body.Close()

	if err != nil {
		handler.ErrorResponse(w, "invalid request - empty body", http.StatusBadRequest, h.logger)
		return
	}

	if err = json.Unmarshal(buf.Bytes(), &body); err != nil {
		handler.ErrorResponse(w, "could not unmarshal request body", http.StatusBadRequest, h.logger)
		return
	}

	entityToUpdate := model.URLEntity{
		OriginalURL: body.OriginalURL,
		ShortURL:    shortURL,
	}
	w.Header().Set("Content-Type", "application/json")
	updatedEntity, err := h.urlService.Update(*h.ctx, entityToUpdate)
	if err != nil {
		if errors.Is(err, urls.ErrURLNotFound) {
			handler.ErrorResponse(w, err.Error(), http.StatusBadRequest, h.logger)
			return
		} else if errors.Is(err, urls.ErrOriginalURLAlreadyExists) {
			responseData := model.URLCreateResponse{
				Result: entityToUpdate.OriginalURL,
			}
			w.WriteHeader(http.StatusConflict)
			resp, err := json.Marshal(responseData)
			if err != nil {
				handler.ErrorResponse(w, fmt.Sprintf("Could not create short URL for %s", body.OriginalURL), http.StatusInternalServerError, h.logger)
				return
			}
			w.Write(resp)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	resp, err := json.Marshal(model.URLUpdateResponse{OriginalURL: updatedEntity})
	if err != nil {
		handler.ErrorResponse(w, "could not marshal request body", http.StatusInternalServerError, h.logger)
		return
	}
	w.Write(resp)
}

func (h *URLHandler) DeleteURLHandler(w http.ResponseWriter, r *http.Request) {
	shortURL := strings.TrimPrefix(r.URL.Path, "/")
	err := h.urlService.Delete(*h.ctx, shortURL)
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		if errors.Is(err, urls.ErrURLNotFound) {
			handler.ErrorResponse(w, fmt.Sprintf("url with short url '%s' not found", shortURL), http.StatusBadRequest, h.logger)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Could not delete the short URL"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *URLHandler) BulkDeleteURLHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		userID, err := getOrCreateUserID(h, w, r)
		if err != nil {
			h.logger.Info("could not get user id from request", zap.Error(err))
		}
		localCtx, cancel := context.WithTimeout(*h.ctx, time.Second*5)
		defer cancel()

		body, err := io.ReadAll(r.Body)
		defer r.Body.Close()

		if err != nil {
			h.logger.Info("could not read request body", zap.Error(err))
		}
		var urlList []string
		err = json.Unmarshal(body, &urlList)
		if err != nil {
			h.logger.Info("failed to unmarshal request body", zap.Error(err))
		}
		var urlsToDelete []model.URLDelete
		for _, url := range urlList {
			urlsToDelete = append(urlsToDelete, model.URLDelete{
				ShortURL:  url,
				CreatedBy: userID,
			})
		}
		err = h.urlService.BulkDelete(localCtx, urlsToDelete)
		defer wg.Done()
	}()
	wg.Wait()
}

func getOrCreateUserID(h *URLHandler, w http.ResponseWriter, r *http.Request) (int, error) {
	var userID int
	var user model.User
	userToken, err := r.Cookie("Authorization")
	if errors.Is(err, http.ErrNoCookie) {
		userID, err = h.userService.GetByIP(*h.ctx, r.RemoteAddr)
		if err != nil && userID == -1 {
			user, err = h.userService.Create(*h.ctx, r.RemoteAddr)
			if err != nil {
				handler.ErrorResponse(w, "could not create user", http.StatusInternalServerError, h.logger)
				return -1, err
			}
			userID = user.ID
		}
		token, err := h.userService.Login(userID, h.cfg)
		if err != nil {
			return -1, fmt.Errorf("could not create token: %w", err)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "Authorization",
			Value:    token,
			Expires:  time.Now().Add(h.cfg.TokenExp),
			HttpOnly: true,
			Path:     "/",
		})
	} else {
		userID = service.GetUserID(userToken.Value, h.cfg)
		if userID == -1 {
			user, err = h.userService.Create(*h.ctx, r.RemoteAddr)
			if err != nil {
				return -1, fmt.Errorf("could not create token: %w", err)
			}
			token, err := h.userService.Login(user.ID, h.cfg)
			if err != nil {
				handler.ErrorResponse(w, err.Error(), http.StatusBadRequest, h.logger)
				return -1, fmt.Errorf("could not create token: %w", err)
			}
			http.SetCookie(w, &http.Cookie{
				Name:     "Authorization",
				Value:    token,
				Expires:  time.Now().Add(h.cfg.TokenExp),
				HttpOnly: true,
				Path:     "/",
			})
		}
	}
	return userID, nil
}

func URLRouter(ctx *context.Context, app *config.App, urlService service.URLServiceInterface, userService service.UserServiceInterface, cfg *config.Config) chi.Router {
	r := chi.NewRouter()

	r.Use(middlewares.PanicRecoverer(app.Logger))
	r.Use(middleware.RealIP)
	r.Use(logger.RequestLoggerMiddleware(app.Logger))
	r.Use(config.GzipMiddleware)

	urlHandler := NewURLHandler(ctx, app, urlService, userService, cfg)

	r.Route("/", func(r chi.Router) {
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Method %s is forbidden", r.Method)))
		})
		r.Post("/api/shorten", urlHandler.ShortenURLHandler)
		r.Post("/api/shorten/batch", urlHandler.BulkCreateURLHandler)
		r.Get("/api/user/urls", urlHandler.GetUserListHandler)
		r.Delete("/api/user/urls", urlHandler.BulkDeleteURLHandler)
		r.Get("/", urlHandler.GetListHandler)
		r.Post("/", urlHandler.CreateURLHandler)
		r.Get("/{id}", urlHandler.GetURLHandler)
		r.Put("/{id}", urlHandler.UpdateURLHandler)
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

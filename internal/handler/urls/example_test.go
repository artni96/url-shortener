package urls

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"time"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/config/db"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/repository/urls"
	"github.com/artni96/url-shortener/internal/repository/users"
	"github.com/artni96/url-shortener/internal/service"
	"go.uber.org/zap"
)

func URLExampleHandler() *URLHandler {
	tempDir := "test"
	testFile := filepath.Join(tempDir, "data.json")
	testLogger := zap.NewNop()

	ctx := context.Background()

	cfg := config.Config{
		ServerAddress:   "localhost:8080",
		ResponseDomain:  "http://localhost:8080",
		FileStoragePath: testFile,
		TokenExp:        time.Hour,
		SecretKey:       "dontshareme",
	}

	auditChan := make(chan model.AuditEntity, 10)

	app := config.App{
		DB:        nil,
		Cfg:       &cfg,
		Logger:    testLogger,
		AuditChan: auditChan,
	}

	testDB, err := db.InitDBConnection(ctx, &app)
	if err == nil {
		app.DB = testDB
	}

	urlInMemoryRepository, err := urls.NewInMemoryURLRepository(&app)
	userInMemoryRepository, err := users.NewInMemoryUserRepository(&app)
	if err != nil {
		log.Fatal(err)
	}
	urlService := service.NewURLService(nil, urlInMemoryRepository, &app)
	userService := service.NewUserService(nil, userInMemoryRepository, &app)
	h := NewURLHandler(&ctx, &app, urlService, userService, &cfg)

	return h
}

func ExampleURLHandler_GetURLHandler() {
	h := URLExampleHandler()
	r := httptest.NewRequest(http.MethodGet, "/KUSwVPQOtfa", nil)
	w := httptest.NewRecorder()
	h.GetURLHandler(w, r)

	fmt.Println(w.Code)
	// Response status: 200
}

func ExampleURLHandler_ShortenURLHandler() {
	h := URLExampleHandler()
	body := strings.NewReader(`{"url":"https://practicum.com"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/shorten", body)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ShortenURLHandler(w, r)

	fmt.Println(w.Code)
	// Response status: 201
}

func ExampleURLHandler_BulkCreateURLHandler() {
	h := URLExampleHandler()
	body := strings.NewReader(`[
    {
        "correlation_id":"1",
        "original_url":"https://yandex1.ru"
    },
    {
        "correlation_id":"2",
        "original_url":"https://yandex2.ru"
    },
        {
        "correlation_id":"3",
        "original_url":"https://yandex3.ru"
    }
	]`)
	r := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", body)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.BulkCreateURLHandler(w, r)

	fmt.Println(w.Code)
	// Response status: 201
}

func ExampleURLHandler_CreateURLHandler() {
	h := URLExampleHandler()
	body := strings.NewReader(`https://yandex.ru`)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateURLHandler(w, r)

	fmt.Println(w.Code)
	// Response status: 201
}

func ExampleURLHandler_GetUserListHandler() {
	h := URLExampleHandler()
	r := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	w := httptest.NewRecorder()
	h.GetUserListHandler(w, r)

	fmt.Println(w.Code)
	// Response status: 200
}

func ExampleURLHandler_BulkDeleteURLHandler() {
	h := URLExampleHandler()
	body := strings.NewReader(`["ddq00Z8_k5", "BLS2jqkJYU", "zsFOp5xfaG"`)
	r := httptest.NewRequest(http.MethodDelete, "/api/user/urls", body)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.BulkDeleteURLHandler(w, r)

	fmt.Println(w.Code)
	// Response status: 202
}

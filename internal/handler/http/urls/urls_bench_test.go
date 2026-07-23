package urls

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/repository/urls"
	"github.com/artni96/url-shortener/internal/repository/users"
	"github.com/artni96/url-shortener/internal/service"
)

func URLTestBenchHandler(b *testing.B) *URLHandler {
	tempDir := b.TempDir()
	testFile := filepath.Join(tempDir, "data.json")
	testLogger := zap.NewNop()

	ctx := b.Context()

	cfg := config.Config{
		ServerAddress:   "localhost:8080",
		ResponseDomain:  "http://localhost:8080",
		FileStoragePath: testFile,
		TokenExp:        time.Hour,
		SecretKey:       "dontshareme",
	}

	//app := config.App{
	//	DB:     nil,
	//	Cfg:    &cfg,
	//	Logger: testLogger,
	//}

	//testDB, err := db.InitDBConnection(ctx, &cfg, testLogger)
	//if err == nil {
	//	app.DB = testDB
	//}

	urlInMemoryRepository, err := urls.NewInMemoryURLRepository(&cfg, testLogger)
	if err != nil {
		b.Fatal(err)
	}
	userInMemoryRepository, err := users.NewInMemoryUserRepository(&cfg, testLogger)
	if err != nil {
		b.Fatal(err)
	}
	urlService := service.NewURLService(nil, urlInMemoryRepository, &cfg, testLogger)
	userService := service.NewUserService(nil, userInMemoryRepository, &cfg, testLogger)
	h := NewURLHandler(&ctx, &cfg, testLogger, urlService, userService, nil)

	return h
}

func BenchmarkURLHandlers(b *testing.B) {
	h := URLTestBenchHandler(b)
	b.Run("BenchmarkShortenURL", func(b *testing.B) {
		reqBody := strings.NewReader(`{"url":"https://google.com"}`)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", reqBody)
			w := httptest.NewRecorder()
			h.ShortenURLHandler(w, req)
		}
	})

	b.Run("BenchmarkCreateURL", func(b *testing.B) {
		reqBody := strings.NewReader(`{"url":"https://google.com"}`)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", reqBody)
			w := httptest.NewRecorder()
			h.CreateURLHandler(w, req)
		}
	})

	b.Run("BenchmarkGetURLHandler", func(b *testing.B) {
		reqToCreate := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("http://test1.com"))
		w := httptest.NewRecorder()
		h.CreateURLHandler(w, reqToCreate)
		shortURL := w.Body.String()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {

			req := httptest.NewRequest(http.MethodGet, shortURL, nil)
			w = httptest.NewRecorder()

			h.GetURLHandler(w, req)
		}
	})

	b.Run("BenchmarkBulkCreateURLHandler", func(b *testing.B) {
		reqBody := strings.NewReader(`[
					{
						"correlation_id":"1",
						"original_url":"http://dqj7vsrbcet.com/ud5irar"
					},
					{
						"correlation_id":"2",
						"original_url":"http://asknw0dlqrb.net/r4ez0dbf"
					}
					]`)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest(http.MethodPost, "/", reqBody)
			w := httptest.NewRecorder()
			h.BulkCreateURLHandler(w, req)

		}
	})

	b.Run("BenchmarkUpdateURLHandler", func(b *testing.B) {
		body := []byte("https://practicum.yandex.ru/")
		testEntityReqBody := strings.NewReader(string(body))
		testEntityReq := httptest.NewRequest(http.MethodPost, "/", testEntityReqBody)
		w := httptest.NewRecorder()
		h.CreateURLHandler(w, testEntityReq)
		testEntityResult := w.Result()
		defer testEntityResult.Body.Close()

		testEntityBody, err := io.ReadAll(testEntityResult.Body)
		if err != nil {
			b.Fatal(err)
		}
		prefix := "http://localhost:8080/"
		shortURL := strings.TrimPrefix(string(testEntityBody), prefix)
		reqBody := strings.NewReader(`{"url":"https://test1.com"}`)
		uri := fmt.Sprintf("/%s", shortURL)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest(http.MethodPut, uri, reqBody)
			w = httptest.NewRecorder()
			h.UpdateURLHandler(w, req)
		}
	})

	b.Run("BenchmarkDeleteURLHandler", func(b *testing.B) {
		reqBody := strings.NewReader(("https://practicum.yandex.ru/"))
		testEntityReq := httptest.NewRequest(http.MethodPost, "/", reqBody)
		w := httptest.NewRecorder()
		h.CreateURLHandler(w, testEntityReq)
		testEntityResult := w.Result()
		testEntityBody, err := io.ReadAll(testEntityResult.Body)
		if err != nil {
			b.Fatal(err)
		}
		prefix := "http://localhost:8080/"
		shortURL := strings.TrimPrefix(string(testEntityBody), prefix)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/%s", shortURL), nil)
			w = httptest.NewRecorder()
			h.DeleteURLHandler(w, req)
		}
	})

	b.Run("BenchmarkGetUserListHandler", func(b *testing.B) {
		token, err := h.userService.BuildJWTString(1, h.cfg)
		if err != nil {
			b.Fatal(err)
		}
		cookie := &http.Cookie{
			Name:    "Authorization",
			Value:   token,
			Path:    "/",
			Expires: time.Now().Add(time.Hour * 2),
		}
		w := httptest.NewRecorder()
		body := []byte("https://practicum.yandex.ru/")
		reqBody := strings.NewReader(string(body))
		newURLRec := httptest.NewRequest(http.MethodPost, "/", reqBody)
		newURLRec.AddCookie(cookie)
		h.CreateURLHandler(w, newURLRec)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w = httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.AddCookie(cookie)
			h.GetUserListHandler(w, req)
		}
	})

	b.Run("BenchmarkBulkDeleteURLHandler", func(b *testing.B) {
		userID := 1
		token, err := h.userService.BuildJWTString(userID, h.cfg)
		if err != nil {
			b.Fatal(err)
		}

		cookie := &http.Cookie{
			Name:    "Authorization",
			Value:   token,
			Path:    "/",
			Expires: time.Now().Add(time.Hour * 2),
		}

		type Result struct {
			status int
			url    string
		}

		urlsNum := 3

		var mockURLsUser1 []model.URLBulkCreateRequest

		for i := range urlsNum {

			mockURLsUser1 = append(mockURLsUser1, model.URLBulkCreateRequest{
				CorrelationID: string(rune(i)),
				OriginalURL:   fmt.Sprintf("%s%d", "https://practicum.yandex.ru/", i),
				CreatedBy:     userID,
			})
		}
		body, err := json.Marshal(mockURLsUser1)
		if err != nil {
			b.Fatal(err)
		}
		bulkCreateReqBody := strings.NewReader(string(body))
		bulkCreateReq := httptest.NewRequest(http.MethodPost, "/", bulkCreateReqBody)
		bulkCreateReq.AddCookie(cookie)

		w1 := httptest.NewRecorder()
		h.BulkCreateURLHandler(w1, bulkCreateReq)
		respBody1 := w1.Result().Body
		defer respBody1.Close()

		bulkCreateReqBodyResp1, err := io.ReadAll(respBody1)
		if err != nil {
			b.Fatal(err)
		}
		var URLList1 []model.URLBulkCreate
		err = json.Unmarshal(bulkCreateReqBodyResp1, &URLList1)
		if err != nil {
			b.Fatal(err)
		}

		var shortURLListWithStatus []Result
		for _, urlData := range URLList1 {
			uri, parseErr := url.Parse(urlData.ShortURL)
			if parseErr != nil {
				b.Fatal(parseErr)
			}
			shortURLListWithStatus = append(shortURLListWithStatus, Result{
				status: http.StatusGone,
				url:    strings.TrimPrefix(uri.Path, "/")},
			)
		}

		var shortURLList []string
		for _, urlData := range shortURLListWithStatus {
			shortURLList = append(shortURLList, urlData.url)
		}
		bodyBytes, err := json.Marshal(shortURLList)
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader(string(bodyBytes)))
			req.AddCookie(cookie)
			w3 := httptest.NewRecorder()
			h.BulkDeleteURLHandler(w3, req)
		}
	})
}

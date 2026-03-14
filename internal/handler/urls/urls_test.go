package urls

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/repository"
	"github.com/artni96/url-shortener/internal/service"
	"github.com/artni96/url-shortener/internal/utility"
	"github.com/stretchr/testify/assert"
)

func TestCreateURLHandler(t *testing.T) {
	cfg := config.Config{
		ServerAddress: "localhost:8080",
		ResponseURL:   "http://localhost:8080",
	}
	urlRepo := repository.NewURLRepository()
	urlService := service.NewURLService(urlRepo)
	h := NewURLHandler(&cfg, urlService)

	type want struct {
		status      int
		contentType string
	}
	type request struct {
		body   []byte
		method string
	}
	tests := []struct {
		name    string
		request request
		want    want
	}{
		{
			name: "success",
			want: want{
				status:      http.StatusCreated,
				contentType: "text/plain",
			},
			request: request{
				method: http.MethodPost,
				body:   []byte("https://practicum.yandex.ru/"),
			},
		},
		{
			name: "empty body",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain",
			},
			request: request{
				method: http.MethodPost,
				body:   []byte{},
			},
		},
		{
			name: "invalid url",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain",
			},
			request: request{
				method: http.MethodPost,
				body:   []byte("practicum.yandex.ru/"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stringBody := strings.NewReader(string(tt.request.body))
			req := httptest.NewRequest(tt.request.method, "/", stringBody)
			w := httptest.NewRecorder()
			h.CreateURLHandler(w, req)
			res := w.Result()
			assert.Equal(t, tt.want.status, res.StatusCode)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			assert.NotEmpty(t, res.Body)
			defer res.Body.Close()
		})
	}
}

func TestGetURLHandler(t *testing.T) {
	cfg := config.Config{
		ServerAddress: "localhost:8080",
		ResponseURL:   "http://localhost:8080",
	}
	urlRepo := repository.NewURLRepository()
	urlService := service.NewURLService(urlRepo)
	h := NewURLHandler(&cfg, urlService)
	type request struct {
		method string
		url    string
	}
	type want struct {
		status      int
		contentType string
		redirectTo  string
	}

	tests := []struct {
		name    string
		request request
		want    want
	}{
		{
			name: "success",
			request: request{
				method: http.MethodGet,
				url:    "/test",
			},
			want: want{
				status:      http.StatusTemporaryRedirect,
				contentType: "text/plain",
				redirectTo:  "http://test.com",
			},
		},
		{
			name: "not found",
			request: request{
				method: http.MethodPost,
				url:    "/test",
			},
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain",
				redirectTo:  "http://test.com",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			reqToCreate := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.want.redirectTo))
			w := httptest.NewRecorder()
			h.CreateURLHandler(w, reqToCreate)
			shortURL := w.Body.String()
			if tt.name == "not found" {
				shortURL = shortURL + "wrong"
			}

			req := httptest.NewRequest(tt.request.method, shortURL, nil)
			w = httptest.NewRecorder()

			h.GetURLHandler(w, req)
			res := w.Result()
			assert.Equal(t, tt.want.status, res.StatusCode)
			if tt.want.status == http.StatusTemporaryRedirect {
				assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
				assert.Equal(t, tt.want.redirectTo, res.Header.Get("Location"))
			}
			defer res.Body.Close()
		})
	}
}

func TestIsURLCorrect(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "correct http",
			url:      "http://test.com",
			expected: true,
		},
		{
			name:     "correct https",
			url:      "https://test.com",
			expected: true,
		},
		{
			name:     "incorrect 1",
			url:      "htp://test.com",
			expected: false,
		},
		{
			name:     "incorrect 2",
			url:      "http:/test.com",
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, isURLCorrect(tt.url), tt.expected)
		})
	}
}

func TestGenerateShortURL(t *testing.T) {
	tests := []struct {
		name     string
		idLength int
	}{
		{
			name:     "success",
			idLength: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := utility.GenerateID(tt.idLength)
			if err != nil {
				t.Errorf("%s", err)
			}
			assert.Equal(t, len(res), tt.idLength)
			assert.NotEmpty(t, res)
			assert.True(t, reflect.TypeOf(res) == reflect.TypeOf("string"))
		})
	}
}

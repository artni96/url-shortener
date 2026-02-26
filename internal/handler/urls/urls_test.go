package urls

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateURLHandler(t *testing.T) {
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
			name: "wrong method",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain",
			},
			request: request{
				method: http.MethodGet,
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stringBody := strings.NewReader(string(tt.request.body))
			req := httptest.NewRequest(tt.request.method, "/", stringBody)
			w := httptest.NewRecorder()
			CreateURLHandler(w, req)
			res := w.Result()
			assert.Equal(t, tt.want.status, res.StatusCode)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			assert.NotEmpty(t, res.Body)
			defer res.Body.Close()
		})
	}
}

func TestGetURLHandler(t *testing.T) {
	type request struct {
		method string
		url    string
	}
	type requestToCreate struct {
		url string
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
			name: "method not allowed",
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
			CreateURLHandler(w, reqToCreate)
			shortURL := w.Body.String()
			if tt.name == "not found" {
				shortURL = shortURL + "wrong"
			}

			fmt.Println(shortURL)
			req := httptest.NewRequest(tt.request.method, shortURL, nil)
			w = httptest.NewRecorder()
			GetURLHandler(w, req)
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

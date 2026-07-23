package urls

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/repository/urls"
	"github.com/artni96/url-shortener/internal/repository/users"
	"github.com/artni96/url-shortener/internal/service"
	"github.com/artni96/url-shortener/internal/utility"
)

func URLTestHandler(t *testing.T) *URLHandler {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "data.json")
	testLogger := zap.NewNop()

	ctx := t.Context()

	cfg := config.Config{
		ServerAddress:   "localhost:8080",
		ResponseDomain:  "http://localhost:8080",
		FileStoragePath: testFile,
		TokenExp:        time.Hour,
		SecretKey:       "dontshareme",
	}

	auditChan := make(chan model.AuditEntity, 10)
	
	urlInMemoryRepository, err := urls.NewInMemoryURLRepository(&cfg, testLogger)
	if err != nil {
		t.Fatal(err)
	}
	userInMemoryRepository, err := users.NewInMemoryUserRepository(&cfg, testLogger)
	if err != nil {
		t.Fatal(err)
	}
	urlService := service.NewURLService(nil, urlInMemoryRepository, &cfg, testLogger)
	userService := service.NewUserService(nil, userInMemoryRepository, &cfg, testLogger)
	h := NewURLHandler(&ctx, &cfg, testLogger, urlService, userService, auditChan)

	return h
}

func TestShortenURL(t *testing.T) {
	h := URLTestHandler(t)

	type want struct {
		contentType string
		status      int
		message     string
	}
	type request struct {
		body   string
		method string
	}
	tests := []struct {
		name    string
		request request
		want    want
	}{
		{
			name: "success",
			request: request{
				body:   `{"url":"https://google.com"}`,
				method: http.MethodPost,
			},
			want: want{
				contentType: "application/json",
				status:      http.StatusCreated,
				message:     `\{\"result\":\"https?://[^\"]+\"\}`,
			},
		},
		{
			name: "invalid url",
			request: request{
				body:   `{"url":"hhtps://google.com"}`,
				method: http.MethodPost,
			},
			want: want{
				contentType: "application/json",
				status:      http.StatusBadRequest,
				message:     `{"error":"url 'hhtps://google.com' is invalid"}`,
			},
		},
		{
			name: "empty body",
			request: request{
				body:   ``,
				method: http.MethodPost,
			},
			want: want{
				contentType: "application/json",
				status:      http.StatusBadRequest,
				message:     `{"error":"could not unmarshal request body"}`,
			},
		},
		{
			name: "check unique urls",
			request: request{
				body:   `{"url":"https://google.com"}`,
				method: http.MethodPost,
			},
			want: want{
				contentType: "application/json",
				status:      http.StatusConflict,
				message:     `\{\"result\":\"https?://[^\"]+\"\}`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := strings.NewReader(tt.request.body)
			req := httptest.NewRequest(tt.request.method, "/api/shorten", reqBody)
			w := httptest.NewRecorder()
			h.ShortenURLHandler(w, req)
			res := w.Result()

			assert.Equal(t, tt.want.status, res.StatusCode)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			strBody, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			if tt.want.status == http.StatusBadRequest {
				assert.JSONEq(t, tt.want.message, string(strBody))
			} else {
				assert.Regexpf(t, tt.want.message, string(strBody), "expected message didn't match")
			}
			assert.NotEmpty(t, res.Body)
			defer res.Body.Close()
		})
	}
}

func TestCreateURLHandler(t *testing.T) {
	h := URLTestHandler(t)

	type want struct {
		status      int
		contentType string
		message     string
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
				message:     `https?://[^\"]+`,
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
		{
			name: "check unique urls",
			want: want{
				status:      http.StatusConflict,
				contentType: "text/plain",
				message:     `https?://[^\"]+`,
			},
			request: request{
				method: http.MethodPost,
				body:   []byte("https://practicum.yandex.ru/"),
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

			if tt.want.status == http.StatusCreated {
				strBody, err := io.ReadAll(res.Body)
				if err != nil {
					t.Fatal(err)
				}
				assert.Regexpf(t, tt.want.message, string(strBody), "expected message didn't match")
			}

			defer res.Body.Close()
		})
	}
}

func TestGetURLHandler(t *testing.T) {
	h := URLTestHandler(t)

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
				redirectTo:  "http://test1.com",
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
				redirectTo:  "http://test2.com",
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
				assert.Equal(t, tt.want.status, res.StatusCode)
				assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
				assert.Equal(t, tt.want.redirectTo, res.Header.Get("Location"))
			}
			defer res.Body.Close()
		})
	}
}

func TestBulkCreateURLHandler(t *testing.T) {
	h := URLTestHandler(t)

	type want struct {
		status      int
		contentType string
		message     string
	}
	type request struct {
		body   string
		method string
	}
	tests := []struct {
		name    string
		request request
		want    want
	}{
		{
			name: "success",
			request: request{
				body: `[
						{
							"correlation_id":"1",
							"original_url":"http://dqj7vsrbcet.com/ud5irar"
						},
						{
							"correlation_id":"2",
							"original_url":"http://asknw0dlqrb.net/r4ez0dbf"
						}
					]`,
				method: http.MethodPost,
			},
			want: want{
				status:      http.StatusCreated,
				contentType: "application/json",
				message:     `\[\{\"correlation_id\":\"[^\"]+\",\"short_url\":\"https?://[^\"]+\"\}(,\{\"correlation_id\":\"[^\"]+\",\"short_url\":\"https?://[^\"]+\"\})*\]`,
			},
		},

		{
			name: "wrong body",
			request: request{
				body: `[
					{
						"correlation_idd":"1",
						"original_url":"http://dqj7vsrbcet.com/ud5irar"
					},
					{
						"correlation_idd":"2",
						"original_url":"http://asknw0dlqrb.net/r4ez0dbf"
					}
				]`,
				method: http.MethodPost,
			},
			want: want{
				status:      http.StatusBadRequest,
				contentType: "application/json",
			},
		},

		{
			name: "duplicated url",
			request: request{
				body: `[
						{
							"correlation_id":"1",
							"original_url":"http://dqj7vsrbcet.com/ud5irar"
						},
						{
							"correlation_id":"2",
							"original_url":"http://dqj7vsrbcet.com/ud5irar"
						}
					]`,
				method: http.MethodPost,
			},
			want: want{
				status:      http.StatusConflict,
				contentType: "application/json",
				message:     `{"error":"duplicated url: http://dqj7vsrbcet.com/ud5irar"}`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := strings.NewReader(tt.request.body)

			req := httptest.NewRequest(tt.request.method, "/", reqBody)
			w := httptest.NewRecorder()
			h.BulkCreateURLHandler(w, req)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.want.status, res.StatusCode)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			strBody, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			assert.Regexpf(t, tt.want.message, string(strBody), "expected message didn't match")
		})
	}
}

func TestGetListHandler(t *testing.T) {
	h := URLTestHandler(t)

	type want struct {
		status      int
		contentType string
		message     string
	}
	type request struct {
		method string
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
			},
			want: want{
				status:      http.StatusOK,
				contentType: "application/json",
				message:     "null",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.request.method, "/", nil)
			w := httptest.NewRecorder()
			h.GetListHandler(w, req)
			res := w.Result()
			defer res.Body.Close()
			assert.Equal(t, tt.want.status, res.StatusCode)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			resBody, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tt.want.message, string(resBody))
		})
	}
}

func TestUpdateURLHandler(t *testing.T) {
	h := URLTestHandler(t)

	type want struct {
		status      int
		contentType string
		message     string
	}
	type request struct {
		method string
		body   string
	}
	tests := []struct {
		name    string
		request request
		want    want
	}{
		{
			name: "success",
			request: request{
				method: http.MethodPut,
				body:   `{"url":"https://google.com"}`,
			},
			want: want{
				status:      http.StatusOK,
				contentType: "application/json",
				message:     `{"url":"https://google.com"}`,
			},
		},
		{
			name: "not found",
			request: request{
				method: http.MethodPut,
				body:   `{"url":"https://test1.com"}`,
			},
			want: want{
				status:      http.StatusBadRequest,
				contentType: "application/json",
				message:     `{"error":"url not found"}`,
			},
		},
		{
			name: "url already exists",
			request: request{
				method: http.MethodPut,
				body:   `{"url":"https://test1.com"}`,
			},
			want: want{
				status:      http.StatusConflict,
				contentType: "application/json",
				message:     `{"result":"https://test1.com"}`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte("https://practicum.yandex.ru/")

			if tt.name == "url already exists" {
				body = []byte("https://test1.com")
			}
			testEntityReqBody := strings.NewReader(string(body))

			testEntityReq := httptest.NewRequest(http.MethodPost, "/", testEntityReqBody)
			w := httptest.NewRecorder()
			h.CreateURLHandler(w, testEntityReq)

			testEntityResult := w.Result()
			assert.Equal(t, http.StatusCreated, testEntityResult.StatusCode)
			defer testEntityResult.Body.Close()
			testEntityBody, err := io.ReadAll(testEntityResult.Body)
			prefix := "http://localhost:8080/"
			shortURL := strings.TrimPrefix(string(testEntityBody), prefix)

			if err != nil {
				t.Fatal(err)
			}

			reqBody := strings.NewReader(tt.request.body)
			uri := fmt.Sprintf("/%s", shortURL)
			if tt.name == "not found" {
				uri = fmt.Sprintf("/%stest", shortURL)
			}
			req := httptest.NewRequest(tt.request.method, uri, reqBody)

			w = httptest.NewRecorder()
			h.UpdateURLHandler(w, req)
			res := w.Result()
			defer res.Body.Close()
			assert.Equal(t, tt.want.status, res.StatusCode)
			resBody, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			assert.JSONEq(t, tt.want.message, string(resBody))
		})
	}
}

func TestDeleteURLHandler(t *testing.T) {
	h := URLTestHandler(t)

	type want struct {
		status      int
		contentType string
		message     string
	}
	type request struct {
		method string
	}
	tests := []struct {
		name    string
		request request
		want    want
	}{
		{
			name: "success",
			request: request{
				method: http.MethodDelete,
			},
			want: want{
				status:      http.StatusNoContent,
				contentType: "application/json",
				message:     "null",
			},
		},
		{
			name: "url not found",
			request: request{
				method: http.MethodDelete,
			},
			want: want{
				status:      http.StatusBadRequest,
				contentType: "application/json",
				message:     "null",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortURL := "123"
			if tt.name == "success" {
				body := []byte("https://practicum.yandex.ru/")
				reqBody := strings.NewReader(string(body))

				testEntityReq := httptest.NewRequest(http.MethodPost, "/", reqBody)
				w := httptest.NewRecorder()
				h.CreateURLHandler(w, testEntityReq)

				testEntityResult := w.Result()
				assert.Equal(t, http.StatusCreated, testEntityResult.StatusCode)
				defer testEntityResult.Body.Close()
				testEntityBody, err := io.ReadAll(testEntityResult.Body)

				prefix := "http://localhost:8080/"
				shortURL = strings.TrimPrefix(string(testEntityBody), prefix)
				if err != nil {
					t.Fatal(err)
				}
			}

			req := httptest.NewRequest(tt.request.method, fmt.Sprintf("/%s", shortURL), nil)
			w := httptest.NewRecorder()
			h.DeleteURLHandler(w, req)
			res := w.Result()
			defer res.Body.Close()
			assert.Equal(t, tt.want.status, res.StatusCode)

		})
	}
}

func TestGetUserListHandler(t *testing.T) {
	h := URLTestHandler(t)

	token, err := h.userService.BuildJWTString(1, h.cfg)
	if err != nil {
		t.Fatal(err)
	}
	cookie := &http.Cookie{
		Name:    "Authorization",
		Value:   token,
		Path:    "/",
		Expires: time.Now().Add(time.Hour * 2),
	}

	type want struct {
		status      int
		contentType string
		message     string
	}
	type request struct {
		method string
		cookie *http.Cookie
	}
	tests := []struct {
		name    string
		request request
		want    want
	}{
		{
			name: "response 200",
			request: request{
				method: http.MethodGet,
				cookie: cookie,
			},
			want: want{
				status:      http.StatusOK,
				contentType: "application/json",
				message:     `\[\{"original_url":"https?://[^"]+","short_url":"https?://localhost:8080/[^"]+"\}\]`,
			},
		},
		{
			name: "response 200",
			request: request{
				method: http.MethodGet,
				cookie: cookie,
			},
			want: want{
				status:      http.StatusOK,
				contentType: "application/json",
				message:     "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.name == "response 200" {
				w1 := httptest.NewRecorder()
				body := []byte("https://practicum.yandex.ru/")
				reqBody := strings.NewReader(string(body))
				newURLRec := httptest.NewRequest(http.MethodPost, "/", reqBody)
				newURLRec.AddCookie(tt.request.cookie)
				h.CreateURLHandler(w1, newURLRec)
				newURLRes := w1.Result()
				defer newURLRes.Body.Close()
			} else if tt.name == "response 204" {
				token2, err := h.userService.BuildJWTString(2, h.cfg)
				if err != nil {
					t.Fatal(err)
				}
				cookie2 := &http.Cookie{
					Name:    "Authorization",
					Value:   token2,
					Path:    "/",
					Expires: time.Now().Add(time.Hour * 2),
				}

				w2 := httptest.NewRecorder()
				body := []byte("https://practicum.yandex.ru/")
				reqBody := strings.NewReader(string(body))
				newURLRec := httptest.NewRequest(http.MethodPost, "/", reqBody)
				newURLRec.AddCookie(cookie2)
				h.CreateURLHandler(w2, newURLRec)
			}
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.request.method, "/", nil)

			req.AddCookie(tt.request.cookie)
			h.GetUserListHandler(w, req)
			res := w.Result()
			defer res.Body.Close()
			assert.Equal(t, tt.want.status, res.StatusCode)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			resBody, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			assert.Regexp(t, tt.want.message, string(resBody))
		})
	}
}

func TestBulkDeleteURLHandler(t *testing.T) {
	h := URLTestHandler(t)
	userID1 := 1

	token1, err := h.userService.BuildJWTString(userID1, h.cfg)
	if err != nil {
		t.Fatal(err)
	}
	cookie1 := &http.Cookie{
		Name:    "Authorization",
		Value:   token1,
		Path:    "/",
		Expires: time.Now().Add(time.Hour * 2),
	}

	userID2 := 2
	token2, err := h.userService.BuildJWTString(userID2, h.cfg)
	if err != nil {
		t.Fatal(err)
	}
	cookie2 := &http.Cookie{
		Name:    "Authorization",
		Value:   token2,
		Path:    "/",
		Expires: time.Now().Add(time.Hour * 2),
	}

	type Result struct {
		status int
		url    string
	}
	type want struct {
		status      int
		contentType string
	}
	type request struct {
		method string
		url    string
	}
	type mocks struct {
		urlsNum int
	}
	tests := []struct {
		name    string
		request request
		want    want
		mocks   mocks
	}{
		{
			name: "success",
			request: request{
				method: http.MethodDelete,
				url:    "https://practicum.yandex.ru/",
			},
			want: want{
				status:      http.StatusAccepted,
				contentType: "application/json",
			},
			mocks: mocks{
				urlsNum: 3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ссылки созданные пользователем 1
			var mockURLsUser1 []model.URLBulkCreateRequest

			for i := range tt.mocks.urlsNum {

				mockURLsUser1 = append(mockURLsUser1, model.URLBulkCreateRequest{
					CorrelationID: string(rune(i)),
					OriginalURL:   fmt.Sprintf("%s%d", tt.request.url, i),
					CreatedBy:     userID1,
				})
			}
			body, err := json.Marshal(mockURLsUser1)
			if err != nil {
				t.Fatal(err)
			}
			bulkCreateReqBody := strings.NewReader(string(body))
			bulkCreateReq := httptest.NewRequest(http.MethodPost, "/", bulkCreateReqBody)
			bulkCreateReq.AddCookie(cookie1)

			w1 := httptest.NewRecorder()
			h.BulkCreateURLHandler(w1, bulkCreateReq)
			respBody1 := w1.Result().Body
			defer respBody1.Close()

			bulkCreateReqBodyResp1, err := io.ReadAll(respBody1)
			if err != nil {
				t.Fatal(err)
			}
			var URLList1 []model.URLBulkCreate
			err = json.Unmarshal(bulkCreateReqBodyResp1, &URLList1)
			if err != nil {
				t.Fatal(err)
			}

			var shortURLListWithStatus []Result
			for _, urlData := range URLList1 {
				uri, parseErr := url.Parse(urlData.ShortURL)
				if parseErr != nil {
					t.Fatal(parseErr)
				}
				shortURLListWithStatus = append(shortURLListWithStatus, Result{
					status: http.StatusGone,
					url:    strings.TrimPrefix(uri.Path, "/")},
				)
			}

			assert.Equal(t, http.StatusCreated, w1.Result().StatusCode)
			assert.Equal(t, tt.want.contentType, w1.Result().Header.Get("Content-Type"))

			// ссылки созданные пользователем 2
			var mockURLsUser2 []model.URLBulkCreateRequest

			for i := range tt.mocks.urlsNum {

				mockURLsUser2 = append(mockURLsUser2, model.URLBulkCreateRequest{
					CorrelationID: string(rune(i)),
					OriginalURL:   fmt.Sprintf("%s%d", tt.request.url, i),
					CreatedBy:     userID2,
				})
			}
			body, err = json.Marshal(mockURLsUser2)
			if err != nil {
				t.Fatal(err)
			}
			bulkCreateReqBody2 := strings.NewReader(string(body))
			bulkCreateReq = httptest.NewRequest(http.MethodPost, "/", bulkCreateReqBody2)
			bulkCreateReq.AddCookie(cookie2)

			w2 := httptest.NewRecorder()
			h.BulkCreateURLHandler(w2, bulkCreateReq)
			respBody2 := w2.Result().Body
			defer respBody2.Close()

			bulkCreateReqBodyResp2, err := io.ReadAll(respBody2)
			if err != nil {
				t.Fatal(err)
			}
			var URLList2 []model.URLBulkCreate
			err = json.Unmarshal(bulkCreateReqBodyResp2, &URLList2)
			if err != nil {
				t.Fatal(err)
			}

			for _, urlData := range URLList2 {
				uri, parseErr := url.Parse(urlData.ShortURL)
				if parseErr != nil {
					t.Fatal(parseErr)
				}
				shortURLListWithStatus = append(shortURLListWithStatus, Result{
					status: http.StatusTemporaryRedirect,
					url:    strings.TrimPrefix(uri.Path, "/")},
				)
			}

			assert.Equal(t, http.StatusCreated, w2.Result().StatusCode)
			assert.Equal(t, tt.want.contentType, w2.Result().Header.Get("Content-Type"))

			var shortURLList []string
			for _, urlData := range shortURLListWithStatus {
				shortURLList = append(shortURLList, urlData.url)
			}

			bodyBytes, err := json.Marshal(shortURLList)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader(string(bodyBytes)))
			req.AddCookie(cookie1)
			w3 := httptest.NewRecorder()
			h.BulkDeleteURLHandler(w3, req)
			assert.Equal(t, tt.want.status, w3.Result().StatusCode)
			assert.Equal(t, tt.want.contentType, w3.Result().Header.Get("Content-Type"))

			for _, urlData := range shortURLListWithStatus {
				w := httptest.NewRecorder()
				req1 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s", urlData.url), nil)
				h.GetURLHandler(w, req1)
				assert.Equal(t, urlData.status, w.Result().StatusCode, urlData.url)
			}
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
			res, err := utility.GenerateShortURL(tt.idLength)
			if err != nil {
				t.Errorf("%s", err)
			}
			assert.Equal(t, len(res), tt.idLength)
			assert.NotEmpty(t, res)
			assert.True(t, reflect.TypeOf(res) == reflect.TypeOf("string"))
		})
	}
}

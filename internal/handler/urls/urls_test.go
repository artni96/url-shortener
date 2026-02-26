package urls

import (
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
			name:    "success",
			want:    want{status: http.StatusCreated, contentType: "text/plain"},
			request: request{method: http.MethodPost, body: []byte("https://practicum.yandex.ru/")},
		},
		{
			name:    "wrong method",
			want:    want{status: http.StatusBadRequest, contentType: "text/plain"},
			request: request{method: http.MethodGet, body: []byte("https://practicum.yandex.ru/")},
		},
		{
			name:    "empty body",
			want:    want{status: http.StatusBadRequest, contentType: "text/plain"},
			request: request{method: http.MethodPost, body: []byte{}},
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
		})
	}
}

//func TestGetURLHandler(t *testing.T) {
//	type args struct {
//		w http.ResponseWriter
//		r *http.Request
//	}
//	tests := []struct {
//		name string
//		args args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			GetURLHandler(tt.args.w, tt.args.r)
//		})
//	}
//}
//
//func Test_generateID(t *testing.T) {
//	type args struct {
//		length int
//	}
//	tests := []struct {
//		name    string
//		args    args
//		want    string
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := generateID(tt.args.length)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("generateID() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if got != tt.want {
//				t.Errorf("generateID() got = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}

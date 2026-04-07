package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestPingHandler(t *testing.T) {
	ctx := t.Context()
	logger := zap.NewNop()
	cfg := config.Config{
		ServerAddress: "localhost:8080",
	}
	app := config.App{
		Cfg:    &cfg,
		Logger: logger,
	}
	h := NewHealthCheckHandler(&ctx, &app)

	type request struct {
		method string
	}
	type want struct {
		status int
	}
	tests := []struct {
		name    string
		request request
		want    want
	}{
		{
			name: "unsuccessful ping request",
			request: request{
				method: "GET",
			},
			want: want{
				status: http.StatusInternalServerError,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.request.method, "/ping", nil)
			w := httptest.NewRecorder()
			h.PingHandler(w, req)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.want.status, res.StatusCode)
		})
	}
}

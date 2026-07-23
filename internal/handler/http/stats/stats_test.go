package stats

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/config/db"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/repository/stats"
	"github.com/artni96/url-shortener/internal/repository/urls"
	"github.com/artni96/url-shortener/internal/repository/users"
	"github.com/artni96/url-shortener/internal/service"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func testServices(app config.App, t *testing.T) (service.URLServiceInterface, service.UserServiceInterface, service.StatsServiceInterface) {
	urlInMemoryRepository, err := urls.NewInMemoryURLRepository(&app)
	if err != nil {
		t.Fatal(err)
	}
	urlService := service.NewURLService(nil, urlInMemoryRepository, &app)

	userInMemoryRepository, err := users.NewInMemoryUserRepository(&app)
	if err != nil {
		t.Fatal(err)
	}
	userService := service.NewUserService(nil, userInMemoryRepository, &app)

	statsDBRepository, err := stats.NewDBStatsRepository(&app)
	if err != nil {
		t.Fatal(err)
	}
	statsService := service.NewStatsService(statsDBRepository, &app, userService, urlService)
	return urlService, userService, statsService
}

func useFixture(ctx context.Context, t *testing.T, testIdx int, urlService service.URLServiceInterface, userService service.UserServiceInterface) {
	testUser, err := userService.Create(ctx, fmt.Sprintf("127.0.0.%d", testIdx))
	assert.NoError(t, err)
	urlData := model.URLCreateRequest{
		OriginalURL: fmt.Sprintf("http://example%d.com", testIdx),
		CreatedBy:   testUser.ID,
	}
	_, err = urlService.Create(ctx, urlData, "http://localhost.com")
	assert.NoError(t, err)
}

func TestGetStats(t *testing.T) {
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

	urlService, userService, statsService := testServices(app, t)

	h := NewStatsHandler(app.Logger, statsService)

	type want struct {
		message string
	}
	tests := []struct {
		name    string
		want    want
		testIdx int
	}{
		{
			name: "success (empty storage)",
			want: want{
				message: `{"urls":0,"users":0}`,
			},
			testIdx: 0,
		},
		{
			name: "success (one url - one user)",
			want: want{
				message: `{"urls":1,"users":1}`,
			},
			testIdx: 1,
		},
		{
			name: "success (two urls - two users)",
			want: want{
				message: `{"urls":2,"users":2}`,
			},
			testIdx: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.testIdx > 0 {
				useFixture(ctx, t, tt.testIdx, urlService, userService)
			}
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			h.GetStatsHandler(w, req)
			res := w.Result()
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
			assert.Equal(t, tt.want.message, string(resBody))

		})
	}
}

package logger

import (
	"net/http"

	"go.uber.org/zap"
)

var Logger *zap.Logger = zap.NewNop()

func InitLogger(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}
	cfg := zap.NewDevelopmentConfig()
	cfg.Level = lvl
	logger, err := cfg.Build()
	if err != nil {
		return err
	}
	Logger = logger
	return nil
}

func RequestLogger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Logger.Info("request started",
			zap.String("method", r.Method),
			zap.String("url", r.URL.Path),
		)
		h.ServeHTTP(w, r)
	})
}

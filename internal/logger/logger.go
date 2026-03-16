package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

var Logger *zap.SugaredLogger = zap.NewNop().Sugar()

type (
	responseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(status int) {
	r.ResponseWriter.WriteHeader(status)
	r.responseData.status = status
}

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
	sugar := logger.Sugar()
	Logger = sugar
	return nil
}

func RequestLogger(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}
		h.ServeHTTP(&lw, r)

		duration := time.Since(start)

		Logger.Infoln(
			"uri", r.RequestURI,
			"method", r.Method,
			"duration", duration,
			"response_status", responseData.status,
			"response_size", responseData.size,
		)
	}
	return http.HandlerFunc(logFn)
}

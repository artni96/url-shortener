package logger

import (
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	levelMap := map[string]zapcore.Level{
		"debug": zapcore.DebugLevel,
		"info":  zapcore.InfoLevel,
		"warn":  zapcore.WarnLevel,
		"error": zapcore.ErrorLevel,
		"fatal": zapcore.FatalLevel,
	}

	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	logFileConfig := zap.NewDevelopmentEncoderConfig()
	fileEncoder := zapcore.NewJSONEncoder(logFileConfig)

	logFile, err := os.OpenFile("./internal/logger/shortener.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	fileOut := zapcore.NewCore(fileEncoder, zapcore.AddSync(logFile), levelMap[level])
	stdOut := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), levelMap[level])

	loggerCore := zapcore.NewTee(fileOut, stdOut)
	logger := zap.New(loggerCore, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
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

package middlewares

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"

	"go.uber.org/zap"
)

type errorResponse struct {
	Error string `json:"error"`
}

func PanicRecoverer(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Info("got panic", zap.String("error message", string(debug.Stack())))
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					respMessage := errorResponse{
						Error: fmt.Sprintf("panic recovered: %v", recovered),
					}
					resp, err := json.Marshal(respMessage)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					w.Write(resp)

					return
				}
			}()

			next.ServeHTTP(w, r)

		}
		return http.HandlerFunc(fn)
	}
}

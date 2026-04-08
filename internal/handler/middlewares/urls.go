package middlewares

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"

	"go.uber.org/zap"
)

func PanicRecoverer(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Info("got panic",
						zap.String("error message", fmt.Sprintf("panic recovered: %v\n", recovered)),
						zap.String("call stack", string(debug.Stack())),
					)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					resp, _ := json.Marshal(struct {
						Error string `json:"error"`
					}{
						Error: "Internal Server Error",
					})
					w.Write(resp)

					return
				}
			}()

			next.ServeHTTP(w, r)

		}
		return http.HandlerFunc(fn)
	}
}

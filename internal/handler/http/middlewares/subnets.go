package middlewares

import (
	"net"
	"net/http"

	"go.uber.org/zap"
)

// CheckIP checks if user ip belongs to app trusted subnet ot not.
func CheckIP(logger *zap.Logger, appSubnet string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			strIP := r.Header.Get("X-Real-IP")
			w.Header().Set("Content-Type", "application/json")
			if strIP == "" {
				w.WriteHeader(http.StatusForbidden)
			}
			ip := net.ParseIP(strIP)
			if ip == nil {
				w.WriteHeader(http.StatusForbidden)
			}

			_, subnet, err := net.ParseCIDR(appSubnet)
			if err != nil {
				logger.Error("Error parsing CIDR", zap.Error(err))
				w.WriteHeader(http.StatusForbidden)
			}
			if subnet.Contains(ip) {
				next.ServeHTTP(w, r)
			}
			w.WriteHeader(http.StatusForbidden)
		}
		return http.HandlerFunc(fn)
	}
}

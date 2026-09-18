package global

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/navire-dev/navire/core/internal/utils"
	"github.com/navire-dev/navire/shared/logger"
)

// Log writes structured access logs using a single source of truth for client IP.
// trustProxy should be true when the app runs behind a trusted proxy/tunnel.
func Log(log logger.Logger, trustProxy bool, trustedProxies *utils.IPMatcher) func(http.Handler) http.Handler {
	// Low-noise endpoints to skip (tweak as you like).
	skip := map[string]struct{}{
		"/healthz": {},
		"/metrics": {},
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := skip[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			// Proxy-aware client IP
			ip := utils.ClientIP(r, trustProxy, trustedProxies)

			// Route pattern (e.g. /users/{id}) for low-cardinality logs
			route := ""
			if rc := chi.RouteContext(r.Context()); rc != nil {
				route = rc.RoutePattern()
			}

			status := ww.Status()
			fields := []zap.Field{
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("route", route),
				zap.Int("status", status),
				zap.Int("bytes", ww.BytesWritten()),
				zap.Duration("duration", time.Since(start)),
				zap.String("ip", ip),
				zap.String("host", r.Host),
				zap.String("proto", r.Proto),
				zap.Bool("tls", r.TLS != nil),
				zap.String("user_agent", r.UserAgent()),
				zap.String("request_id", middleware.GetReqID(r.Context())),
			}

			switch {
			case status >= 500:
				log.Error("http_request", fields...)
			case status >= 400:
				log.Warn("http_request", fields...)
			default:
				log.Info("http_request", fields...)
			}
		})
	}
}

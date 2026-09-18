package global

import (
	"net/http"

	"github.com/navire-dev/navire/core/internal/utils"
)

// AllowOnlyIPs allows only specific IPs/CIDRs. If the list is empty, it does NOT filter (passthrough).
// trustProxy should be true when running behind a trusted reverse proxy/tunnel (e.g., cloudflared).
func AllowOnlyCIDRS(allowed []string, trustProxy bool, trustedProxies *utils.IPMatcher) func(http.Handler) http.Handler {
	m := utils.NewIPMatcher(allowed)
	if m.IsEmpty() {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := utils.ClientIP(r, trustProxy, trustedProxies)
			if !m.Allow(ip) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

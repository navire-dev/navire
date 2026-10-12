package global

import (
	"net/http"

	"github.com/navire-dev/navire/core/internal/clientip"
)

// AllowOnlyIPs allows only specific IPs/CIDRs. If the list is empty, it does NOT filter (passthrough).
// trustProxy should be true when running behind a trusted reverse proxy/tunnel (e.g., cloudflared).
func AllowOnlyCIDRS(allowed []string, trustProxy bool, trustedProxies *clientip.IPMatcher) func(http.Handler) http.Handler {
	m := clientip.NewIPMatcher(allowed)
	if m.IsEmpty() {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientip.ClientIP(r, trustProxy, trustedProxies)
			if !m.Allow(ip) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

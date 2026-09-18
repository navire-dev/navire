package readyz

import (
	"net/http"
)

// Readyz is a minimal readiness probe.
// It only returns 200 OK and "ready": true in JSON.
func Readyz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ready":true}`))
	}
}

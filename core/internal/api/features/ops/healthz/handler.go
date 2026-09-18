package healthz

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/navire-dev/navire/core/internal/api/deps"
)

type healthzResponse struct {
	Status        string  `json:"status"`
	UptimeSeconds float64 `json:"uptime_seconds"`
	Version       string  `json:"version,omitempty"`
	Commit        string  `json:"commit,omitempty"`
	BuildDate     string  `json:"build_date,omitempty"`
	GoVersion     string  `json:"go_version,omitempty"`
}

func Healthz(d deps.Deps) http.HandlerFunc {
	start := d.StartTime
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(healthzResponse{
			Status:        "ok",
			Version:       d.Version,
			Commit:        d.Commit,
			BuildDate:     d.BuildDate,
			GoVersion:     d.GoVersion,
			UptimeSeconds: time.Since(start).Seconds(),
		})
	}
}

package httpx

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/mayday-team/server/internal/observability"
)

type healthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Uptime    int64  `json:"uptime"`
	Timestamp string `json:"timestamp"`
}

// NewHealthHandler returns the GET /health handler.
func NewHealthHandler(metrics *observability.Metrics) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body := healthResponse{
			Status:    "ok",
			Service:   "mayday-server",
			Uptime:    metrics.UptimeSeconds(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		_ = json.NewEncoder(w).Encode(body)
	}
}

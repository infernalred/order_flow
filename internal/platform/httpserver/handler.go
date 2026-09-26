package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync/atomic"
)

func NewHandler(logger *slog.Logger, readiness *atomic.Bool) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /live", live)
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		ready(w, r, readiness)
	})

	return requestMetadata(
		requestLogging(logger, mux),
	)
}

func live(w http.ResponseWriter, _ *http.Request) {
	writeHealthResponse(w, "live", http.StatusOK)
}

func ready(w http.ResponseWriter, _ *http.Request, readiness *atomic.Bool) {
	if readiness.Load() {
		writeHealthResponse(w, "ready", http.StatusOK)
	} else {
		writeHealthResponse(w, "not_ready", http.StatusServiceUnavailable)
	}
}

func writeHealthResponse(w http.ResponseWriter, status string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(healthResponse{
		Status: status,
	})
}

type healthResponse struct {
	Status string `json:"status"`
}

package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func NewHandler(logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /live", live)
	mux.HandleFunc("GET /ready", ready)

	return requestMetadata(
		requestLogging(logger, mux),
	)
}

func live(w http.ResponseWriter, _ *http.Request) {
	writeHealthResponse(w, "alive")
}

func ready(w http.ResponseWriter, _ *http.Request) {
	writeHealthResponse(w, "ready")
}

func writeHealthResponse(w http.ResponseWriter, status string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(healthResponse{
		Status: status,
	})
}

type healthResponse struct {
	Status string `json:"status"`
}

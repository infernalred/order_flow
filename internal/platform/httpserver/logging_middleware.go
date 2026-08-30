package httpserver

import (
	"log/slog"
	"net/http"
	"time"
)

func requestLogging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()

		response := &statusResponseWriter{ResponseWriter: w}

		next.ServeHTTP(response, r)

		logger.InfoContext(
			r.Context(),
			"HTTP request completed",
			slog.String("request_id", requestIDFromContext(r.Context())),
			slog.String("correlation_id", correlationIDFromContext(r.Context())),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", response.StatusCode()),
			slog.Int64(
				"duration_ms",
				time.Since(startedAt).Milliseconds(),
			),
		)
	})
}

func (w *statusResponseWriter) WriteHeader(statusCode int) {
	if w.statusCode != 0 {
		return
	}

	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *statusResponseWriter) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(b)
}

func (w *statusResponseWriter) StatusCode() int {
	if w.statusCode == 0 {
		return http.StatusOK
	}

	return w.statusCode
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

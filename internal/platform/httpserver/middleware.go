package httpserver

import (
	"context"
	"crypto/rand"
	"net/http"
	"strings"
)

const (
	requestIDHeader     = "X-Request-ID"
	correlationIDHeader = "X-Correlation-ID"
)

type contextKey uint8

const (
	requestIDKey contextKey = iota
	correlationIDKey
)

func requestMetadata(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get(requestIDHeader))

		if requestID == "" {
			requestID = rand.Text()
		}

		correlationID := strings.TrimSpace(r.Header.Get(correlationIDHeader))

		if correlationID == "" {
			correlationID = requestID
		}

		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		ctx = context.WithValue(ctx, correlationIDKey, correlationID)

		w.Header().Set(requestIDHeader, requestID)
		w.Header().Set(correlationIDHeader, correlationID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey).(string)

	return requestID
}

func correlationIDFromContext(ctx context.Context) string {
	correlationID, _ := ctx.Value(correlationIDKey).(string)

	return correlationID
}

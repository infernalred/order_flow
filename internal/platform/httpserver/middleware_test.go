package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestMetadata(t *testing.T) {
	tests := []struct {
		name          string
		requestID     string
		correlationID string
	}{
		{
			name:          "both provided",
			requestID:     "request-123",
			correlationID: "correlation-456",
		},
		{
			name: "neither provided",
		},
		{
			name:      "only request ID",
			requestID: "request-123",
		},
		{
			name:          "only correlation ID",
			correlationID: "correlation-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotRequestID string
			var gotCorrelationID string

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotRequestID = requestIDFromContext(r.Context())
				gotCorrelationID = correlationIDFromContext(r.Context())
				w.WriteHeader(http.StatusNoContent)
			})

			handler := requestMetadata(next)

			request := httptest.NewRequest(http.MethodGet, "/", nil)

			if tt.requestID != "" {
				request.Header.Set(requestIDHeader, tt.requestID)
			}

			if tt.correlationID != "" {
				request.Header.Set(correlationIDHeader, tt.correlationID)
			}

			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			if resp.Code != http.StatusNoContent {
				t.Errorf("status code = %v want %v", resp.Code, http.StatusNoContent)
			}

			if tt.requestID != "" && gotRequestID != tt.requestID {
				t.Errorf("request ID = %v want %v", gotRequestID, tt.requestID)
			}

			if gotRequestID == "" {
				t.Errorf("requestID not found in context")
			}

			if tt.correlationID != "" {
				if gotCorrelationID != tt.correlationID {
					t.Errorf("correlationID = %q, want %q", gotCorrelationID, tt.correlationID)
				}
			} else if gotCorrelationID != gotRequestID {
				t.Errorf("correlationID = %q, want %q", gotCorrelationID, gotRequestID)
			}

			if actual := resp.Header().Get(requestIDHeader); actual != gotRequestID {
				t.Errorf("requestID = %q, want %q", actual, gotRequestID)
			}

			if actual := resp.Header().Get(correlationIDHeader); actual != gotCorrelationID {
				t.Errorf("correlationID = %q, want %q", actual, gotCorrelationID)
			}
		})
	}
}

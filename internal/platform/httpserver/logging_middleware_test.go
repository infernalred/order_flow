package httpserver

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestLogging(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	handler := requestMetadata(
		requestLogging(logger, next),
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/orders?token=must-not-be-logged",
		nil,
	)
	request.Header.Set(requestIDHeader, "request-123")
	request.Header.Set(correlationIDHeader, "correlation-456")

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Errorf(
			"status code = %d, want %d",
			response.Code,
			http.StatusCreated,
		)
	}

	var record struct {
		Level         string `json:"level"`
		Message       string `json:"msg"`
		RequestID     string `json:"request_id"`
		CorrelationID string `json:"correlation_id"`
		Method        string `json:"method"`
		Path          string `json:"path"`
		Status        int    `json:"status"`
		DurationMS    *int64 `json:"duration_ms"`
	}

	if err := json.NewDecoder(&output).Decode(&record); err != nil {
		t.Fatalf("decode HTTP log: %v", err)
	}

	if record.Level != "INFO" {
		t.Errorf("log level = %q, want %q", record.Level, "INFO")
	}
	if record.Message != "HTTP request completed" {
		t.Errorf(
			"log message = %q, want %q",
			record.Message,
			"HTTP request completed",
		)
	}
	if record.RequestID != "request-123" {
		t.Errorf("request ID = %q, want %q", record.RequestID, "request-123")
	}
	if record.CorrelationID != "correlation-456" {
		t.Errorf(
			"correlation ID = %q, want %q",
			record.CorrelationID,
			"correlation-456",
		)
	}
	if record.Method != http.MethodPost {
		t.Errorf("method = %q, want %q", record.Method, http.MethodPost)
	}
	if record.Path != "/orders" {
		t.Errorf("path = %q, want %q", record.Path, "/orders")
	}
	if record.Status != http.StatusCreated {
		t.Errorf(
			"logged status = %d, want %d",
			record.Status,
			http.StatusCreated,
		)
	}
	if record.DurationMS == nil {
		t.Fatal("duration_ms is missing from HTTP log")
	}
	if *record.DurationMS < 0 {
		t.Errorf("duration_ms = %d, want a non-negative value", *record.DurationMS)
	}
}

func TestStatusResponseWriter(t *testing.T) {
	tests := []struct {
		name       string
		write      func(http.ResponseWriter)
		wantStatus int
	}{
		{
			name: "explicit status",
			write: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusCreated)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "implicit status from body",
			write: func(w http.ResponseWriter) {
				_, _ = w.Write([]byte("response body"))
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "no response written",
			write:      func(http.ResponseWriter) {},
			wantStatus: http.StatusOK,
		},
		{
			name: "first status wins",
			write: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusCreated)
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			underlying := httptest.NewRecorder()
			response := &statusResponseWriter{
				ResponseWriter: underlying,
			}

			tt.write(response)

			if actual := response.StatusCode(); actual != tt.wantStatus {
				t.Errorf("status code = %d, want %d", actual, tt.wantStatus)
			}

			if actual := underlying.Result().StatusCode; actual != tt.wantStatus {
				t.Errorf(
					"underlying status code = %d, want %d",
					actual,
					tt.wantStatus,
				)
			}
		})
	}
}

package httpserver

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestNewHandler(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantHealth string
	}{
		{
			name:       "live",
			method:     http.MethodGet,
			path:       "/live",
			wantStatus: http.StatusOK,
			wantHealth: "live",
		},
		{
			name:       "ready",
			method:     http.MethodGet,
			path:       "/ready",
			wantStatus: http.StatusOK,
			wantHealth: "ready",
		},
		{
			name:       "unsupported method",
			method:     http.MethodPost,
			path:       "/live",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "unknown path",
			method:     http.MethodGet,
			path:       "/unknown",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.path, nil)
			resp := httptest.NewRecorder()

			var readiness atomic.Bool
			readiness.Store(true)

			handler := NewHandler(newDiscardLogger(), &readiness)
			handler.ServeHTTP(resp, request)

			if resp.Code != tt.wantStatus {
				t.Errorf("NewHandler returned wrong status code: got %v want %v", resp.Code, tt.wantStatus)
			}

			if tt.wantHealth == "" {
				return
			}

			contentType := resp.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
			}

			var body struct {
				Status string `json:"status"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode response body: %v", err)
			}

			if body.Status != tt.wantHealth {
				t.Errorf("body status = %q, want %q", body.Status, tt.wantHealth)
			}
		})
	}
}

func TestWriteHeaderOrder(t *testing.T) {
	resp1 := httptest.NewRecorder()
	resp1.WriteHeader(http.StatusServiceUnavailable)
	_, err := resp1.Write([]byte("Hello"))
	if err != nil {
		t.Fatalf("write response: %v", err)
	}

	if resp1.Result().StatusCode != http.StatusServiceUnavailable {
		t.Errorf("resp1 status code = %v, want %v", resp1.Result().StatusCode, http.StatusServiceUnavailable)
	}

	resp2 := httptest.NewRecorder()
	_, err2 := resp2.Write([]byte("Hello"))
	resp2.WriteHeader(http.StatusServiceUnavailable)
	if err2 != nil {
		t.Fatalf("write response: %v", err2)
	}
	if resp2.Result().StatusCode != http.StatusOK {
		t.Errorf("resp2 status code = %v, want %v", resp2.Result().StatusCode, http.StatusOK)
	}
}

func TestCheckStatus(t *testing.T) {
	var readiness atomic.Bool
	readiness.Store(true)

	request := httptest.NewRequest(http.MethodGet, "/live", nil)
	liveBefore := httptest.NewRecorder()

	handler := NewHandler(newDiscardLogger(), &readiness)
	handler.ServeHTTP(liveBefore, request)

	request2 := httptest.NewRequest(http.MethodGet, "/ready", nil)
	readyBefore := httptest.NewRecorder()

	handler.ServeHTTP(readyBefore, request2)

	if liveBefore.Result().StatusCode != http.StatusOK {
		t.Errorf("liveBefore status code = %v, want %v", liveBefore.Result().StatusCode, http.StatusOK)
	}

	if readyBefore.Result().StatusCode != http.StatusOK {
		t.Errorf("readyBefore status code = %v, want %v", readyBefore.Result().StatusCode, http.StatusOK)
	}

	var body struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(readyBefore.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if body.Status != "ready" {
		t.Errorf("body status = %q, want %q", body.Status, "ready")
	}

	if err := json.NewDecoder(liveBefore.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if body.Status != "live" {
		t.Errorf("body status = %q, want %q", body.Status, "live")
	}

	readiness.Store(false)

	liveAfter := httptest.NewRecorder()
	handler.ServeHTTP(liveAfter, request)

	readyAfter := httptest.NewRecorder()
	handler.ServeHTTP(readyAfter, request2)

	if liveAfter.Result().StatusCode != http.StatusOK {
		t.Errorf("liveAfter status code = %v, want %v", liveAfter.Result().StatusCode, http.StatusOK)
	}

	if readyAfter.Result().StatusCode != http.StatusServiceUnavailable {
		t.Errorf("readyAfter status code = %v, want %v", readyAfter.Result().StatusCode, http.StatusServiceUnavailable)
	}

	if err := json.NewDecoder(liveAfter.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if body.Status != "live" {
		t.Errorf("body status = %q, want %q", body.Status, "live")
	}

	if err := json.NewDecoder(readyAfter.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if body.Status != "not_ready" {
		t.Errorf("body status = %q, want %q", body.Status, "not_ready")
	}
}

func newDiscardLogger() *slog.Logger {
	return slog.New(
		slog.NewTextHandler(io.Discard, nil),
	)
}

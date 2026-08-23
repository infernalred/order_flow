package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
			wantHealth: "alive",
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

			handler := NewHandler()
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

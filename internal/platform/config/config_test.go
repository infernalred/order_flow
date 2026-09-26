package config

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	setValidEnvironment(t)

	actual, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if actual.Environment != "local" {
		t.Errorf("Environment = %q, want %q", actual.Environment, "local")
	}

	if actual.HTTP.Address != ":8080" {
		t.Errorf("HTTP.Address = %q, want %q", actual.HTTP.Address, ":8080")
	}

	if actual.HTTP.ReadTimeout != 5*time.Second {
		t.Errorf(
			"HTTP.ReadTimeout = %v, want %v",
			actual.HTTP.ReadTimeout,
			5*time.Second,
		)
	}
	if actual.HTTP.WriteTimeout != 10*time.Second {
		t.Errorf(
			"HTTP.WriteTimeout = %v, want %v",
			actual.HTTP.WriteTimeout,
			10*time.Second,
		)
	}
	if actual.HTTP.ShutdownTimeout != 10*time.Second {
		t.Errorf(
			"HTTP.ShutdownTimeout = %v, want %v",
			actual.HTTP.ShutdownTimeout,
			10*time.Second,
		)
	}
	if actual.HTTP.IdleTimeout != 60*time.Second {
		t.Errorf(
			"HTTP.IdleTimeout = %v, want %v",
			actual.HTTP.IdleTimeout,
			60*time.Second,
		)
	}
	if actual.HTTP.ReadHeaderTimeout != 5*time.Second {
		t.Errorf(
			"HTTP.ReadHeaderTimeout = %v, want %v",
			actual.HTTP.ReadHeaderTimeout,
			5*time.Second,
		)
	}

	if actual.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want %q", actual.Log.Level, "info")
	}
}

func TestLoadEnvironment(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("APP_ENV", "test")

	actual, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if actual.Environment != "test" {
		t.Errorf("Environment = %q, want %q", actual.Environment, "test")
	}
}

func TestLoadMissingEnvironmentVariables(t *testing.T) {
	keys := []string{
		"APP_ENV",
		"HTTP_ADDR",
		"HTTP_READ_TIMEOUT",
		"HTTP_WRITE_TIMEOUT",
		"HTTP_SHUTDOWN_TIMEOUT",
		"HTTP_IDLE_TIMEOUT",
		"HTTP_READ_HEADER_TIMEOUT",
		"LOG_LEVEL",
	}

	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			setValidEnvironment(t)
			if err := os.Unsetenv(key); err != nil {
				t.Fatalf("Unsetenv(%q) error = %v", key, err)
			}

			_, err := Load()
			if !errors.Is(err, ErrMissingEnvironmentVariable) {
				t.Fatalf(
					"Load() error = %v, want %v",
					err,
					ErrMissingEnvironmentVariable,
				)
			}
			if !strings.Contains(err.Error(), key) {
				t.Errorf("Load() error = %q, want variable name %q", err, key)
			}
		})
	}
}

func TestLoadInvalidEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{
			name:  "HTTP address",
			key:   "HTTP_ADDR",
			value: "not-an-address",
		},
		{
			name:  "HTTP port outside valid range",
			key:   "HTTP_ADDR",
			value: ":70000",
		},
		{
			name:  "read timeout",
			key:   "HTTP_READ_TIMEOUT",
			value: "not-a-duration",
		},
		{
			name:  "write timeout",
			key:   "HTTP_WRITE_TIMEOUT",
			value: "not-a-duration",
		},
		{
			name:  "shutdown timeout",
			key:   "HTTP_SHUTDOWN_TIMEOUT",
			value: "not-a-duration",
		},
		{
			name:  "idle timeout",
			key:   "HTTP_IDLE_TIMEOUT",
			value: "not-a-duration",
		},
		{
			name:  "read header timeout",
			key:   "HTTP_READ_HEADER_TIMEOUT",
			value: "not-a-duration",
		},
		{
			name:  "zero duration",
			key:   "HTTP_READ_TIMEOUT",
			value: "0s",
		},
		{
			name:  "negative duration",
			key:   "HTTP_WRITE_TIMEOUT",
			value: "-1s",
		},
		{
			name:  "zero duration",
			key:   "HTTP_IDLE_TIMEOUT",
			value: "0s",
		},
		{
			name:  "zero duration",
			key:   "HTTP_READ_HEADER_TIMEOUT",
			value: "0s",
		},
		{
			name:  "log level",
			key:   "LOG_LEVEL",
			value: "verbose",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setValidEnvironment(t)
			t.Setenv(tt.key, tt.value)

			_, err := Load()
			if !errors.Is(err, ErrInvalidEnvironmentVariable) {
				t.Fatalf(
					"Load() error = %v, want %v",
					err,
					ErrInvalidEnvironmentVariable,
				)
			}
			if !strings.Contains(err.Error(), tt.key) {
				t.Errorf("Load() error = %q, want variable name %q", err, tt.key)
			}
		})
	}
}

func setValidEnvironment(t *testing.T) {
	t.Helper()

	t.Setenv("APP_ENV", "local")
	t.Setenv("HTTP_ADDR", ":8080")
	t.Setenv("HTTP_READ_TIMEOUT", "5s")
	t.Setenv("HTTP_WRITE_TIMEOUT", "10s")
	t.Setenv("HTTP_SHUTDOWN_TIMEOUT", "10s")
	t.Setenv("HTTP_IDLE_TIMEOUT", "60s")
	t.Setenv("HTTP_READ_HEADER_TIMEOUT", "5s")
	t.Setenv("LOG_LEVEL", "info")
}

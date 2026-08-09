package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestNewValidateLevel(t *testing.T) {
	_, err := New(os.Stdout, "log/slog")

	if err == nil {
		t.Fatal("New() error = nil, want an error")
	}

	if !strings.Contains(err.Error(), "parse log level") {
		t.Errorf("Log level = %q", err)
	}
}

func TestNewWritesJSON(t *testing.T) {
	var output bytes.Buffer

	logger, err := New(&output, "info")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	logger.Info(
		"application initialized",
		slog.String("environment", "test"),
	)

	var record struct {
		Level       string `json:"level"`
		Message     string `json:"msg"`
		Environment string `json:"environment"`
	}

	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("log is not valid JSON: %v\nlog: %s", err, output.String())
	}

	if record.Level != "INFO" {
		t.Errorf("Level = %q, want %q", record.Level, "INFO")
	}
	if record.Message != "application initialized" {
		t.Errorf("Message = %q, want %q", record.Message, "application initialized")
	}
	if record.Environment != "test" {
		t.Errorf("Environment = %q, want %q", record.Environment, "test")
	}

	output.Reset()
	logger.Debug("debug message")

	if output.Len() != 0 {
		t.Errorf("debug log was written at info level: %s", output.String())
	}
}

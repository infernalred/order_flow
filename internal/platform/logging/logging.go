package logging

import (
	"fmt"
	"io"
	"log/slog"
)

// New returns a JSON logger configured with the provided level.
func New(output io.Writer, levelText string) (*slog.Logger, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(levelText)); err != nil {
		return nil, fmt.Errorf("parse log level %q: %w", levelText, err)
	}

	handler := slog.NewJSONHandler(output, &slog.HandlerOptions{
		Level: level,
	})
	return slog.New(handler), nil
}

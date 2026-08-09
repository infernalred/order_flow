package main

import (
	"log/slog"
	"os"

	"github.com/infernalred/order_flow/internal/platform/config"
	"github.com/infernalred/order_flow/internal/platform/logging"
)

func main() {
	bootstrapLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load()
	if err != nil {
		bootstrapLogger.Error("load configuration", slog.Any("error", err))
		os.Exit(1)
	}

	logger, err := logging.New(os.Stdout, cfg.Log.Level)
	if err != nil {
		bootstrapLogger.Error("create logger", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info(
		"application initialized",
		slog.String("environment", cfg.Environment),
		slog.String("address", cfg.HTTP.Address))
}

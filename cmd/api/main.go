package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"

	"github.com/infernalred/order_flow/internal/platform/config"
	"github.com/infernalred/order_flow/internal/platform/httpserver"
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

	handler := httpserver.NewHandler(logger)

	server := &http.Server{
		Addr:         cfg.HTTP.Address,
		Handler:      handler,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	logger.Info(
		"HTTP server starting",
		slog.String("address", cfg.HTTP.Address),
		slog.String("read_timeout", server.ReadTimeout.String()),
		slog.String("write_timeout", server.WriteTimeout.String()),
		slog.String("idle_timeout", server.IdleTimeout.String()),
	)

	if err := server.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		logger.Error(
			"HTTP server failed",
			slog.Any("error", err),
		)
		os.Exit(1)
	}
}

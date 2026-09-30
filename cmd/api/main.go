package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

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

	var readiness atomic.Bool
	readiness.Store(true)
	handler := httpserver.NewHandler(logger, &readiness)

	server := &http.Server{
		Addr:              cfg.HTTP.Address,
		Handler:           handler,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
	}

	logger.Info(
		"HTTP server starting",
		slog.String("address", cfg.HTTP.Address),
		slog.String("read_timeout", server.ReadTimeout.String()),
		slog.String("write_timeout", server.WriteTimeout.String()),
		slog.String("idle_timeout", server.IdleTimeout.String()),
		slog.String("header_timeout", server.ReadHeaderTimeout.String()),
	)

	signalCtx, stopSignals := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stopSignals()

	serverError := make(chan error, 1)
	go func() {
		serverError <- server.ListenAndServe()
	}()

	select {
	case err = <-serverError:
		if err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server failed", slog.Any("error", err))
			os.Exit(1)
		}
		logger.Info("HTTP server stopped")
	case <-signalCtx.Done():
		stopSignals()
		logger.Info("HTTP server stopping")
		readiness.Store(false)
		ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
		defer cancel()
		shutdownErr := server.Shutdown(ctx)
		listenErr := <-serverError
		if shutdownErr != nil || (listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed)) {
			_ = server.Close()
			if shutdownErr != nil {
				logger.Error("HTTP server shutdown failed", slog.Any("error", shutdownErr))
			}
			if listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
				logger.Error("HTTP server listen failed", slog.Any("error", listenErr))
			}
			os.Exit(1)
		}

		logger.Info("HTTP server stopped successfully")
	}
}

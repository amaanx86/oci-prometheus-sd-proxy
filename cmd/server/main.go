package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amaanx86/oci-prometheus-sd-proxy/internal/config"
	"github.com/amaanx86/oci-prometheus-sd-proxy/internal/discovery"
	"github.com/amaanx86/oci-prometheus-sd-proxy/internal/server"
)

// version is set at build time via ldflags:
//
//	go build -ldflags "-X main.version=v1.2.3"
var version = "dev"

func main() {
	// Structured JSON logging for machine-parseable output in production.
	// Swap NewJSONHandler for NewTextHandler during local development if preferred.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("oci-prometheus-sd-proxy starting", "version", version)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Build the in-memory target cache and kick off background refresh.
	cache := discovery.NewCache(cfg)
	cache.Start(ctx)

	srv := server.New(cfg, cache)

	// Run the HTTP server in a goroutine so we can wait for shutdown signals.
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Block until SIGINT / SIGTERM or the server itself errors out.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("received shutdown signal", "signal", sig)
	case err := <-serverErr:
		slog.Error("server error", "error", err)
	}

	cancel() // stop background discovery goroutine

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}

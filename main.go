package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/acidsailor/ts3afkmover/config"
	"github.com/acidsailor/ts3afkmover/internal/idle"
	"github.com/acidsailor/ts3afkmover/internal/ts3"
)

func main() {
	slog.SetDefault(
		slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
		})),
	)

	cfg, err := config.New()
	if err != nil {
		slog.Error(
			"error initializing config",
			slog.String("err", err.Error()),
		)
		os.Exit(1)
	}

	client, err := ts3.New(cfg)
	if err != nil {
		slog.Error(
			"error initializing ts3 client",
			slog.String("err", err.Error()),
		)
		os.Exit(1)
	}

	mover := idle.New(cfg, client)

	ctx, stop := signal.NotifyContext(
		context.Background(), syscall.SIGINT, syscall.SIGTERM,
	)
	defer stop()

	ticker := time.NewTicker(cfg.TickInterval())
	defer ticker.Stop()

	slog.InfoContext(ctx, "starting application")

	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "shutting down")
			return
		case <-ticker.C:
			mover.MoveIdleClients(ctx)
		}
	}
}

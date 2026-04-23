package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jsabo/tsentry/internal/client"
	"github.com/jsabo/tsentry/internal/config"
	"github.com/jsabo/tsentry/internal/watcher"
)

// Set by GoReleaser via -ldflags at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cfg, err := config.Parse()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "tsentry %s (%s, %s)\n", version, commit[:min(len(commit), 7)], date)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Stdin mode: no Teleport connection needed, reads JSON events from stdin.
	if cfg.Stdin {
		w := watcher.NewStdin(cfg)
		if err := w.RunStdin(ctx, os.Stdin); err != nil {
			slog.Error("stdin error", "error", err)
			os.Exit(1)
		}
		return
	}

	tc, err := client.New(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to Teleport", "error", err)
		os.Exit(1)
	}
	defer tc.Close()

	w := watcher.New(tc, cfg)
	if err := w.Run(ctx); err != nil {
		slog.Error("watcher exited with error", "error", err)
		os.Exit(1)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Command server is the main entrypoint for the ReadIT SSH forum server.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/sahas/readit/internal/config"
	"github.com/sahas/readit/internal/db"
	internalssh "github.com/sahas/readit/internal/ssh"
)

func main() {
	// Force 24-bit TrueColor profile so Lip Gloss default renderer outputs ANSI escape codes
	// even when the server daemon process is started headless under systemd without an interactive TTY.
	lipgloss.SetColorProfile(termenv.TrueColor)

	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// --- Context with graceful shutdown --------------------------------
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// --- Logger --------------------------------------------------------
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// --- Configuration -------------------------------------------------
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger.Info("configuration loaded",
		"ssh_addr", cfg.Address(),
		"host_key", cfg.HostKeyPath,
	)

	// --- Database pool -------------------------------------------------
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	logger.Info("database pool established")

	// --- SSH server ----------------------------------------------------
	srv, err := internalssh.NewServer(cfg, pool, logger)
	if err != nil {
		return err
	}

	logger.Info("starting ReadIT SSH server", "addr", cfg.Address())
	return internalssh.ListenAndServe(ctx, srv, logger)
}

package ssh

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/charmbracelet/ssh"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sahas/readit/internal/config"
	"github.com/sahas/readit/internal/tui"
)

// NewServer creates a configured wish SSH server.
func NewServer(cfg *config.Config, pool *pgxpool.Pool, logger *slog.Logger) (*ssh.Server, error) {
	// teaHandler returns the Bubble Tea model and program options per session.
	teaHandler := func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
		pubKey := sess.PublicKey()
		fingerprint := ""
		if pubKey != nil {
			fingerprint = Fingerprint(pubKey)
		}

		logger.Info("new session",
			"user", sess.User(),
			"remote", sess.RemoteAddr().String(),
			"fingerprint", fingerprint,
		)

		model := tui.NewModel(sess.Context(), pool, fingerprint, logger)
		opts := append(bubbletea.MakeOptions(sess), tea.WithAltScreen())
		return model, opts
	}

	srv, err := wish.NewServer(
		wish.WithAddress(cfg.Address()),
		wish.WithHostKeyPath(cfg.HostKeyPath),
		wish.WithPublicKeyAuth(func(_ ssh.Context, key ssh.PublicKey) bool {
			// Accept all public keys — identity is derived from the key itself.
			// The user record is looked up/created via the fingerprint at session start.
			return true
		}),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating SSH server: %w", err)
	}

	return srv, nil
}

// ListenAndServe starts the SSH server and blocks until the context is cancelled.
func ListenAndServe(ctx context.Context, srv *ssh.Server, logger *slog.Logger) error {
	errCh := make(chan error, 1)

	go func() {
		logger.Info("SSH server listening", "addr", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("SSH server error: %w", err)
	case <-ctx.Done():
		logger.Info("shutting down SSH server")
		if err := srv.Shutdown(ctx); err != nil {
			return fmt.Errorf("SSH server shutdown: %w", err)
		}
		return nil
	}
}

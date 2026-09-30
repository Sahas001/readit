package ssh

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/muesli/termenv"
	"golang.org/x/net/netutil"

	"github.com/sahas/readit/internal/config"
	"github.com/sahas/readit/internal/sanitize"
	"github.com/sahas/readit/internal/tui"
)

// NewServer creates a configured wish SSH server.
func NewServer(cfg *config.Config, pool *pgxpool.Pool, logger *slog.Logger) (*ssh.Server, error) {
	// Force TrueColor profile on Lip Gloss default renderer so all styling emits 24-bit ANSI colors.
	lipgloss.SetColorProfile(termenv.TrueColor)

	// teaHandler returns the Bubble Tea model and program options per session.
	teaHandler := func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
		pubKey := sess.PublicKey()
		if pubKey == nil {
			wish.Fatalln(sess, "Error: Public key authentication is required to access ReadIT.")
			return nil, nil
		}
		fingerprint := Fingerprint(pubKey)

		sanitizedUser := sanitize.SingleLine(sess.User())
		if sanitizedUser == "" {
			sanitizedUser = "anonymous"
		}

		logger.Info("new session",
			"user", sanitizedUser,
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
		wish.WithIdleTimeout(15*time.Minute),
		wish.WithMaxTimeout(2*time.Hour),
		wish.WithPublicKeyAuth(func(_ ssh.Context, key ssh.PublicKey) bool {
			// Accept all public keys — identity is derived from the key itself.
			// The user record is looked up/created via the fingerprint at session start.
			return true
		}),
		wish.WithMiddleware(
			bubbletea.MiddlewareWithColorProfile(teaHandler, termenv.TrueColor),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating SSH server: %w", err)
	}

	return srv, nil
}

// ListenAndServe starts the SSH server with a connection-limiting listener (max 100 concurrent connections)
// and blocks until the context is cancelled.
func ListenAndServe(ctx context.Context, srv *ssh.Server, logger *slog.Logger) error {
	l, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("binding SSH listener: %w", err)
	}
	defer l.Close()

	// Bound concurrent TCP connections to 100 to prevent pre-auth file descriptor and socket buffer exhaustion
	limitedListener := netutil.LimitListener(l, 100)

	errCh := make(chan error, 1)

	go func() {
		logger.Info("SSH server listening", "addr", srv.Addr, "max_conns", 100)
		if err := srv.Serve(limitedListener); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("SSH server error: %w", err)
	case <-ctx.Done():
		logger.Info("shutting down SSH server, draining active sessions")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("SSH server shutdown: %w", err)
		}
		return nil
	}
}


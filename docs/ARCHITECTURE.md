# Architecture

ReadIT is a single-binary SSH server that hosts a Reddit-style terminal forum. Users connect with `ssh -p 2222 forum.domain.com`, and the server renders a full Bubble Tea TUI inside each SSH PTY session. There is no HTTP layer — the server communicates directly with PostgreSQL over an internal connection pool.

## System overview

```
┌─────────────────────────────────────────────────────────┐
│                    ReadIT Server                        │
│                                                         │
│  ┌───────────┐    ┌──────────┐    ┌──────────────────┐  │
│  │   Wish    │───▶│  Auth    │───▶│  Bubble Tea TUI  │  │
│  │ SSH Server│    │Middleware│    │  (per session)   │  │
│  │  :2222    │    │          │    │                  │  │
│  └───────────┘    └──────────┘    └────────┬─────────┘  │
│                                            │            │
│                                   ┌────────▼──────────┐ │
│                                   │  pgxpool (shared) │ │
│                                   └────────┬──────────┘ │
│                                            │            │
└────────────────────────────────────────────┼────────────┘
                                             │
                                    ┌────────▼──────────┐
                                    │  PostgreSQL 16    │
                                    │  (citext, goose)  │
                                    └───────────────────┘
```

## Component responsibilities

### `cmd/server/main.go`

The entrypoint performs the following in sequence:

1. Registers `SIGINT` and `SIGTERM` handlers via `signal.NotifyContext`.
2. Initializes a structured `slog.Logger`.
3. Loads configuration from environment variables.
4. Creates and pings a `pgxpool.Pool`.
5. Constructs the Wish SSH server with middleware.
6. Blocks on `ListenAndServe` until the context is cancelled.

On shutdown, the SSH server drains active sessions before the pool closes.

### `internal/config`

Reads configuration from environment variables with sensible defaults. No external configuration library is used — `os.Getenv` with fallbacks keeps the dependency tree minimal.

| Variable | Default | Purpose |
|---|---|---|
| `READIT_DATABASE_URL` | `postgres://readit:readit_dev@localhost:5432/readit?sslmode=disable` | PostgreSQL connection string |
| `READIT_SSH_HOST` | `0.0.0.0` | SSH listen address |
| `READIT_SSH_PORT` | `2222` | SSH listen port |
| `READIT_HOST_KEY_PATH` | `.ssh/host_key` | Path to the SSH host key (auto-generated on first run) |
| `READIT_LOG_LEVEL` | `info` | Log verbosity |

### `internal/db`

Owns the connection pool (`pool.go`) and the sqlc-generated query layer (`sqlc/`).

The pool is configured with production-grade settings:

- **Max connections:** 25
- **Min connections:** 5
- **Max lifetime:** 1 hour
- **Max idle time:** 30 minutes
- **Health check period:** 1 minute

### `internal/ssh`

Contains two files:

- **`auth.go`** — Computes the SHA256 fingerprint of an SSH public key, matching the format of `ssh-keygen -lf`.
- **`server.go`** — Constructs the Wish server with the middleware chain and implements graceful shutdown.

### `internal/tui`

The Bubble Tea application, split across four files:

- **`model.go`** — Root `tea.Model` with the state machine, async commands, and per-view update handlers.
- **`views.go`** — View renderers for each state (loading, onboarding, board list, post list, error).
- **`keymap.go`** — Vim-style key bindings using `bubbles/key`.
- **`styles.go`** — Lipgloss theme with a Reddit-inspired color palette.

## Data flow

The following sequence describes what happens when a user connects:

```
Client SSH → Wish accepts connection
           → Public key auth callback (accept all)
           → activeterm.Middleware checks PTY
           → bubbletea.Middleware creates tea.Program
           → teaHandler extracts fingerprint, creates Model
           → Model.Init() → lookupUserCmd → DB query
           → User found? → load boards → render board list
           → User not found? → render onboarding form
           → User interacts → Update() dispatches by view state
           → DB queries run as tea.Cmd (async)
           → View() renders current state to SSH PTY
```

## Concurrency model

- Each SSH session gets its own `tea.Program` running in its own goroutine.
- All sessions share a single `pgxpool.Pool`, which handles connection-level concurrency internally.
- Database queries run as `tea.Cmd` functions — they execute in a background goroutine managed by Bubble Tea and return messages to the `Update` loop.
- The `context.Context` passed to each model is the session context, which is cancelled when the SSH session disconnects.

## Error handling

Errors follow a consistent pattern:

1. Database commands return `errMsg` on failure.
2. The `Update` loop catches `errMsg` and transitions to `viewError`.
3. The error view renders the message and offers a quit action.
4. All errors are wrapped with `fmt.Errorf("context: %w", err)` for traceability.

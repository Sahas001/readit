# Development Guide

This document outlines local environment setup, testing procedures, migration management, and development workflows for the ReadIT terminal forum.

---

## Prerequisites

Before starting, install the following tools:

1. **Go 1.23+**: [golang.org/dl](https://golang.org/dl/)
2. **Docker & Docker Compose**: [docker.com](https://www.docker.com/)
3. **Goose** (database migrations CLI):
   ```bash
   go install github.com/pressly/goose/v3/cmd/goose@latest
   ```
4. **sqlc** (type-safe SQL compiler for Go):
   ```bash
   go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
   ```
5. **OpenSSH Client**: `ssh` command line utility.

---

## Quickstart

Clone and boot the application locally in under 2 minutes:

```bash
# 1. Start the PostgreSQL 16 container
make up

# 2. Run initial Goose migrations
make migrate-up

# 3. Build and launch the SSH server on :2222
make run
```

In a separate terminal window, connect via SSH:

```bash
make ssh-test
# or directly:
ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -p 2222 localhost
```

---

## Makefile Reference

The project includes an explicit `Makefile` orchestrating common commands:

| Command | Action |
|---|---|
| `make up` | Starts PostgreSQL container and polls healthcheck until ready. |
| `make down` | Shuts down Docker Compose services. |
| `make migrate-up` | Applies all pending Goose migrations in `migrations/`. |
| `make migrate-down` | Rolls back the most recent Goose migration. |
| `make migrate-create NAME=<name>` | Generates a new timestamped migration SQL file. |
| `make sqlc` | Runs `sqlc generate` to compile queries in `sqlc/queries/` to Go code in `internal/db/sqlc/`. |
| `make build` | Builds the server binary to `bin/readit-server`. |
| `make run` | Builds and executes the server locally. |
| `make ssh-test` | Connects to `localhost:2222` with host key checks disabled. |
| `make clean` | Removes compiled binaries in `bin/`. |
| `make help` | Lists all available targets with descriptions. |

---

## Database Migrations Workflow

Database migrations are stored in `migrations/*.sql` using Goose syntax:

```sql
-- +goose Up
-- SQL statements for applying the migration

-- +goose Down
-- SQL statements for rolling back the migration
```

### Adding a New Migration

1. Create a migration file:
   ```bash
   make migrate-create NAME=add_post_flairs
   ```
2. Edit the newly created file in `migrations/`.
3. Apply the migration:
   ```bash
   make migrate-up
   ```
4. Update or add corresponding queries in `sqlc/queries/`.
5. Recompile SQL queries to Go types:
   ```bash
   make sqlc
   ```

---

## Working with `sqlc`

Queries are defined in pure SQL inside `sqlc/queries/`:

* `users.sql` — User profiles and SSH public key lookups.
* `boards.sql` — Board listings and retrieval.
* `posts.sql` — Paginated post retrieval (`new` and `top`) and creation.
* `comments.sql` — Recursive CTE comment trees.
* `votes.sql` — Upserting votes and recalculating scores.

### Annotations Guide

* `-- name: FunctionName :one`: Query returning a single row.
* `-- name: FunctionName :many`: Query returning a slice of rows.
* `-- name: FunctionName :exec`: Query returning only an error without rows.
* `-- name: FunctionName :execrows`: Query returning number of affected rows.

Whenever you change any query in `sqlc/queries/*.sql` or schema in `migrations/*.sql`, run:
```bash
make sqlc
```

---

## Coding Conventions & Idioms

1. **No Global State**:
   All dependencies (`pgxpool.Pool`, `slog.Logger`, `db.Queries`) must be explicitly passed into constructors (`NewServer`, `NewModel`).
2. **Context Propagation**:
   Every database query must accept `ctx context.Context`. For SSH sessions, always propagate `sess.Context()`.
3. **Explicit Error Wrapping**:
   Wrap errors with actionable context using `fmt.Errorf("doing action: %w", err)`.
4. **Vim Navigation by Default**:
   All interactive lists should support both arrows and `j`/`k` navigation, plus standard shortcuts (`enter` select, `esc` back, `q` quit).
5. **Screen Restoration**:
   Always enable alternate screen buffer (`tea.WithAltScreen()`) to ensure no terminal residue remains after disconnection.

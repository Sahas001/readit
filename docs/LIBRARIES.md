# Library Reference & Project Roadmap

This guide details each third-party dependency in ReadIT, why it was chosen over alternatives, how it fits into the overall architecture, and a structured roadmap for building out the forum features.

---

## 1. Library Reference

### `charmbracelet/wish` & `golang.org/x/crypto/ssh`

* **What it does**: 
  `wish` is an SSH server engine designed specifically for building interactive, terminal-native applications. It wraps Go's low-level `golang.org/x/crypto/ssh` primitives, providing a composable middleware pipeline (analogous to HTTP middleware like `chi` or `gin`) for PTY allocation, session logging, access control, and Bubble Tea lifecycle management.
* **Why we use it**: 
  Writing a bare `crypto/ssh` server from scratch requires managing raw terminal modes, window size change signals (`SIGWINCH`), pseudo-terminal (PTY) emulation, and sub-channel multiplexing manually. `wish` abstracts these complexities while retaining full control over authentication hooks.
* **Role in ReadIT**: 
  Listens on port `2222`, executes public-key authentication, computes the SHA256 fingerprint from `gossh.PublicKey`, verifies PTY presence via `activeterm.Middleware()`, and instantiates a Bubble Tea program for every incoming connection.

---

### `charmbracelet/bubbletea`

* **What it does**: 
  A framework based on The Elm Architecture (TEA) for building terminal user interfaces (TUIs) in Go. It operates on three primitives: **Model** (state), **Update** (event handling and state transitions), and **View** (rendering).
* **Why we use it**: 
  Unlike imperative terminal libraries (such as `tview` or `termbox-go`) where developers manually track cursor positioning and redraw dirty screen buffers, Bubble Tea promotes a declarative, unidirectional data flow. This eliminates UI race conditions and simplifies state management.
* **Role in ReadIT**: 
  Drives the entire client-facing interface. Each SSH session runs an isolated `tea.Program`. Database operations run as asynchronous `tea.Cmd` tasks that post `tea.Msg` events back into the event loop.

---

### `charmbracelet/lipgloss`

* **What it does**: 
  A CSS-like styling library for the terminal. It provides composable primitives for foreground and background colors, bold/italic text, borders, margins, padding, alignment, and multi-column joins.
* **Why we use it**: 
  Manual ANSI escape sequence handling is error-prone, hard to maintain, and fragile across different terminal emulators. Lipgloss automatically detects terminal color capabilities (TrueColor, ANSI 256, ANSI 16) and handles string measurement and line wrapping correctly (including Unicode/East Asian width).
* **Role in ReadIT**: 
  Defines our Reddit-themed design tokens (`#FF4500` orange, upvote/downvote hues, subtle borders), formats selected list items, draws status bars, and centers modal prompts via `lipgloss.Place`.

---

### `charmbracelet/bubbles`

* **What it does**: 
  A collection of pre-built, reusable UI components for Bubble Tea (text inputs, text areas, viewports, progress bars, tables, paginators, spinners).
* **Why we use it**: 
  Implementing interactive text fields that support cursor blinking, text selection, deletion, and arrow key navigation inside a terminal is non-trivial. Bubbles provides hardened implementations conforming directly to `tea.Model`.
* **Role in ReadIT**: 
  * `bubbles/textinput`: Captures handles during user onboarding.
  * `bubbles/key`: Declarative, configurable key bindings and automatic help text generation.
  * `bubbles/textarea` *(Upcoming)*: Multi-line editor for creating posts and comment replies.
  * `bubbles/viewport` *(Upcoming)*: Scrollable containers for long post bodies and comment threads.

---

### `charmbracelet/glamour`

* **What it does**: 
  A stylesheet-driven Markdown renderer for the terminal, transforming raw markdown text into styled ANSI terminal output.
* **Why we use it**: 
  Reddit forums rely heavily on Markdown formatting (bold, italic, code blocks, lists, blockquotes). Glamour renders Markdown with syntax highlighting and responsive wrapping directly in the terminal PTY.
* **Role in ReadIT**: 
  Used in `viewPostDetail` to render post bodies and complex comment descriptions with automatic word wrapping to the user's terminal width.

---

### `jackc/pgx/v5` & `pgxpool`

* **What it does**: 
  The premier pure-Go PostgreSQL driver and toolkit. `pgxpool` provides concurrent connection pooling, automatic reconnection, and health checks.
* **Why we use it**: 
  Standard `database/sql` operates through an abstraction layer that limits PostgreSQL-specific features. `pgx` offers superior performance, native support for PostgreSQL types (arrays, JSON, UUID, composite types, CITEXT), binary protocol encoding, and connection-level health checks.
* **Role in ReadIT**: 
  Manages database connections across all concurrent SSH sessions. A single `*pgxpool.Pool` is initialized in `cmd/server/main.go` and shared across sessions.

---

### `sqlc`

* **What it does**: 
  A SQL compiler that parses raw SQL queries against your database schema and generates fully type-safe, idiomatic Go structs and methods.
* **Why we use it**: 
  * Avoids heavy ORM overhead and opaque query generators (like GORM).
  * Catches SQL syntax errors and schema mismatches at compile time rather than runtime.
  * Generates zero-reflection, fast database access methods.
* **Role in ReadIT**: 
  Compiles SQL in `sqlc/queries/*.sql` against `migrations/` into the `internal/db/sqlc` package. Configured with `sqlc.yaml` to use `pgx/v5` and map `citext` to Go `string`.

---

### `pressly/goose/v3`

* **What it does**: 
  A database migration tool for Go that supports plain `.sql` migration files with forward (`+goose Up`) and rollback (`+goose Down`) scripts.
* **Why we use it**: 
  Plain SQL migrations are transparent, version-controllable, and require no proprietary DSL. Goose can be executed as a standalone CLI tool or embedded directly into a Go binary.
* **Role in ReadIT**: 
  Applies and tracks migrations in `migrations/`, including PostgreSQL extensions (`citext`), table creations, foreign key constraints, composite indexes, and seed boards.

---

## 2. Integration Architecture

Here is how these libraries connect and coordinate in a live session:

```
[Incoming SSH Connection (Client Key)]
                  │
                  ▼
         charmbracelet/wish
    (Validates PTY via activeterm)
    (Calls PublicKeyAuth callback)
                  │
                  ▼
         internal/ssh/auth.go
   (SHA256 Fingerprint computed via golang.org/x/crypto/ssh)
                  │
                  ▼
       charmbracelet/bubbletea
  (Middleware starts tea.Program with tea.WithAltScreen())
                  │
                  ▼
         internal/tui/model.go
      ┌─────────────────────────┐
      │     Model.Init()        │
      └───────────┬─────────────┘
                  │  Dispatches tea.Cmd (Async)
                  ▼
        internal/db/sqlc (Queries)
                  │
                  ▼
         jackc/pgx/v5 (pgxpool)
                  │
                  ▼
             PostgreSQL
```

---

## 3. Recommended Development Approach

Follow this structured phase-by-phase approach to complete the forum:

### Phase 1: Tooling Verification & Real `sqlc` Generation
* **Goal**: Replace the handwritten stub code in `internal/db/sqlc/` with actual generated `sqlc` code.
* **Steps**:
  1. Ensure PostgreSQL is running: `make up`.
  2. Apply migrations: `make migrate-up`.
  3. Install `sqlc`: `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`.
  4. Run `make sqlc`.
  5. Verify clean build: `go build ./...`.

---

### Phase 2: Post Details & Recursive Comment Tree View
* **Goal**: Allow users to press `Enter` on a post in `viewPostList` to open `viewPostDetail`.
* **Steps**:
  1. Add `viewPostDetail` state to `viewState` in `internal/tui/model.go`.
  2. Implement `loadPostDetailCmd(postID int64)`:
     * Fetch post detail using `queries.GetPostByID`.
     * Fetch threaded comments using `queries.GetCommentThreadByPost`.
  3. Format the comment tree:
     * Indent each comment line based on `comment.Depth * 2` spaces.
     * Show `author_handle`, `score▲`, and relative timestamp.
  4. Integrate `charmbracelet/glamour` to render the post's Markdown body with word-wrapping:
     ```go
     renderer, _ := glamour.NewTermRenderer(
         glamour.WithAutoStyle(),
         glamour.WithWordWrap(m.width - 4),
     )
     renderedBody, _ := renderer.Render(post.Body)
     ```

---

### Phase 3: Interactive Voting System
* **Goal**: Allow users to vote on posts and comments using `u` (upvote) and `d` (downvote).
* **Steps**:
  1. In `updatePostList`:
     * When `u` is pressed: dispatch `castPostVoteCmd(postID, 1)`.
     * When `d` is pressed: dispatch `castPostVoteCmd(postID, -1)`.
  2. Command implementation:
     * Execute `queries.UpsertPostVote`.
     * Execute `queries.RecalculatePostScore`.
     * Return a message to reload the current post list or update the item score in-memory.
  3. Apply visual feedback in `internal/tui/styles.go` (e.g., render vote count in orange if upvoted, blue if downvoted).

---

### Phase 4: Content Creation (Submitting Posts & Comments)
* **Goal**: Allow logged-in users to create new posts (`n`) and reply to posts/comments (`r`).
* **Steps**:
  1. Integrate `bubbles/textarea` and `bubbles/textinput` into `Model`:
     * Add `viewNewPost` and `viewNewComment` states.
     * `textinput` for post title, `textarea` for post body.
  2. Add key controls:
     * `Ctrl+S` or `Ctrl+D` to submit.
     * `Esc` to cancel and return to previous view.
  3. Async Command:
     * Execute `queries.CreatePost`.
     * If submitting a comment, execute `queries.CreateComment` followed by `queries.IncrementPostCommentCount`.

---

### Phase 5: Viewport Scrolling & Terminal Polish
* **Goal**: Support smooth scrolling for lengthy discussions that exceed terminal height.
* **Steps**:
  1. Embed `bubbles/viewport` inside `Model` for the detail view.
  2. Update viewport dimensions on `tea.WindowSizeMsg`:
     ```go
     m.viewport.Width = msg.Width
     m.viewport.Height = msg.Height - headerHeight - footerHeight
     ```
  3. Route navigation keys (`k`/`j`, `pageup`/`pagedown`) to `viewport.Update(msg)` when in detail view.

---

### Phase 6: Sorting & Pagination
* **Goal**: Toggle between `New` and `Top` sort orders, and paginate posts.
* **Steps**:
  1. Add `sortMode` enum (`sortNew`, `sortTop`) to `Model`.
  2. Toggle sort with `Tab` key in `viewPostList`.
  3. Call `ListPostsByBoardNew` or `ListPostsByBoardTop` accordingly.
  4. Track pagination offsets (`offset = page * pageSize`) with next (`]` or `l`) and previous (`[` or `h`) page keys.

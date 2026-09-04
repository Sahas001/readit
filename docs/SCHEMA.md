# Database Schema

ReadIT uses PostgreSQL 16 with the `citext` extension for case-insensitive text matching. Migrations are managed with Goose and live in the `migrations/` directory. Queries are defined in `sqlc/queries/` and compiled to Go code with sqlc.

## Entity-relationship diagram

```
┌──────────────┐       ┌──────────────┐       ┌──────────────┐
│    users     │       │    boards    │       │    posts     │
├──────────────┤       ├──────────────┤       ├──────────────┤
│ id        PK │◀──┐   │ id        PK │◀──┐   │ id        PK │
│ pubkey_sha256│   │   │ slug      UK │   │   │ board_id  FK │──▶ boards.id
│ handle    UK │   │   │ title        │   │   │ author_id FK │──▶ users.id
│ bio          │   │   │ description  │   │   │ title        │
│ created_at   │   │   │ created_at   │   │   │ body         │
│ updated_at   │   │   └──────────────┘   │   │ url          │
└──────────────┘   │                      │   │ score        │
        ▲          │                      │   │ comment_cnt  │
        │          │                      │   │ created_at   │
        │          │                      │   │ updated_at   │
        │          │                      │   └──────────────┘
        │          │                      │          ▲
        │          │                      │          │
        │     ┌────┴────────┐        ┌────┴──────────┴──┐
        │     │ post_votes  │        │    comments      │
        │     ├─────────────┤        ├──────────────────┤
        │     │ user_id  PK │        │ id            PK │
        │     │ post_id  PK │        │ post_id       FK │──▶ posts.id
        │     │ direction   │        │ parent_id     FK │──▶ comments.id
        │     │ created_at  │        │ author_id     FK │──▶ users.id
        │     └─────────────┘        │ body             │
        │                            │ score            │
        │     ┌──────────────┐       │ created_at       │
        │     │comment_votes │       │ updated_at       │
        │     ├──────────────┤       └──────────────────┘
        └─────│ user_id   PK │
              │ comment_id PK│
              │ direction    │
              │ created_at   │
              └──────────────┘
```

## Tables

### `users`

Stores user identities. Each user is identified by the SHA256 fingerprint of their SSH public key.

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `BIGINT` | PK, generated always | Auto-incrementing row identifier |
| `pubkey_sha256` | `TEXT` | UNIQUE, NOT NULL | SHA256 fingerprint of the user's SSH public key |
| `handle` | `CITEXT` | UNIQUE, NOT NULL | Case-insensitive display name |
| `bio` | `TEXT` | NOT NULL, default `''` | User biography |
| `created_at` | `TIMESTAMPTZ` | NOT NULL, default `now()` | Account creation timestamp |
| `updated_at` | `TIMESTAMPTZ` | NOT NULL, default `now()` | Last activity timestamp |

**Indexes:**
- `idx_users_pubkey` on `(pubkey_sha256)` — fast lookup during SSH authentication

### `boards`

Top-level discussion categories. Seeded with four default boards on migration.

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `BIGINT` | PK, generated always | Auto-incrementing row identifier |
| `slug` | `CITEXT` | UNIQUE, NOT NULL | URL-style identifier (e.g., `general`, `ask`) |
| `title` | `TEXT` | NOT NULL | Human-readable board name |
| `description` | `TEXT` | NOT NULL, default `''` | Short description shown in the board list |
| `created_at` | `TIMESTAMPTZ` | NOT NULL, default `now()` | Board creation timestamp |

**Seed data:** `general`, `ask`, `show`, `meta`

### `posts`

User-submitted content within a board. Contains denormalized `score` and `comment_count` columns to avoid aggregation queries on every listing.

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `BIGINT` | PK, generated always | Auto-incrementing row identifier |
| `board_id` | `BIGINT` | FK → `boards.id`, CASCADE | Parent board |
| `author_id` | `BIGINT` | FK → `users.id`, CASCADE | Post author |
| `title` | `TEXT` | NOT NULL | Post title |
| `body` | `TEXT` | NOT NULL, default `''` | Post body (supports markdown) |
| `url` | `TEXT` | NOT NULL, default `''` | Optional link |
| `score` | `INT` | NOT NULL, default `0` | Denormalized vote sum |
| `comment_count` | `INT` | NOT NULL, default `0` | Denormalized comment total |
| `created_at` | `TIMESTAMPTZ` | NOT NULL, default `now()` | Post creation timestamp |
| `updated_at` | `TIMESTAMPTZ` | NOT NULL, default `now()` | Last edit timestamp |

**Indexes:**
- `idx_posts_board_created` on `(board_id, created_at DESC)` — reverse-chronological listing
- `idx_posts_board_score` on `(board_id, score DESC, created_at DESC)` — top-score listing
- `idx_posts_author` on `(author_id, created_at DESC)` — user profile page

### `comments`

Threaded comments on posts. The `parent_id` column enables arbitrary nesting depth. Comment threads are fetched with a recursive CTE.

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `BIGINT` | PK, generated always | Auto-incrementing row identifier |
| `post_id` | `BIGINT` | FK → `posts.id`, CASCADE | Parent post |
| `parent_id` | `BIGINT` | FK → `comments.id`, CASCADE, nullable | Parent comment (NULL for top-level) |
| `author_id` | `BIGINT` | FK → `users.id`, CASCADE | Comment author |
| `body` | `TEXT` | NOT NULL | Comment text |
| `score` | `INT` | NOT NULL, default `0` | Denormalized vote sum |
| `created_at` | `TIMESTAMPTZ` | NOT NULL, default `now()` | Comment creation timestamp |
| `updated_at` | `TIMESTAMPTZ` | NOT NULL, default `now()` | Last edit timestamp |

**Indexes:**
- `idx_comments_post` on `(post_id, created_at ASC)` — chronological thread loading
- `idx_comments_parent` on `(parent_id)` — recursive CTE performance
- `idx_comments_author` on `(author_id)` — user profile page

### `post_votes`

Tracks upvotes and downvotes on posts. Each user can vote once per post.

| Column | Type | Constraints | Description |
|---|---|---|---|
| `user_id` | `BIGINT` | PK, FK → `users.id`, CASCADE | Voting user |
| `post_id` | `BIGINT` | PK, FK → `posts.id`, CASCADE | Voted post |
| `direction` | `SMALLINT` | NOT NULL, CHECK `IN (-1, 1)` | `-1` for downvote, `1` for upvote |
| `created_at` | `TIMESTAMPTZ` | NOT NULL, default `now()` | Vote timestamp |

**Indexes:**
- `idx_post_votes_post` on `(post_id)` — score recalculation

### `comment_votes`

Tracks upvotes and downvotes on comments. Identical structure to `post_votes`.

| Column | Type | Constraints | Description |
|---|---|---|---|
| `user_id` | `BIGINT` | PK, FK → `users.id`, CASCADE | Voting user |
| `comment_id` | `BIGINT` | PK, FK → `comments.id`, CASCADE | Voted comment |
| `direction` | `SMALLINT` | NOT NULL, CHECK `IN (-1, 1)` | `-1` for downvote, `1` for upvote |
| `created_at` | `TIMESTAMPTZ` | NOT NULL, default `now()` | Vote timestamp |

**Indexes:**
- `idx_comment_votes_comment` on `(comment_id)` — score recalculation

## Key query patterns

### User upsert on connect

When a user connects via SSH, the server looks up their fingerprint. If the user doesn't exist, the onboarding view collects a handle and upserts a record:

```sql
INSERT INTO users (pubkey_sha256, handle)
VALUES ($1, $2)
ON CONFLICT (pubkey_sha256) DO UPDATE SET updated_at = now()
RETURNING *;
```

### Paginated post listing

Posts support two sort orders — newest and top — both using limit/offset pagination:

```sql
-- Newest
SELECT p.*, u.handle AS author_handle
FROM posts p JOIN users u ON u.id = p.author_id
WHERE p.board_id = $1
ORDER BY p.created_at DESC
LIMIT $2 OFFSET $3;

-- Top
ORDER BY p.score DESC, p.created_at DESC
```

### Recursive comment thread

The full comment tree for a post is fetched in a single query using a recursive CTE. The `depth` and `path` columns enable depth-first rendering with indentation:

```sql
WITH RECURSIVE thread AS (
    SELECT c.*, u.handle, 0 AS depth, ARRAY[c.id] AS path
    FROM comments c JOIN users u ON u.id = c.author_id
    WHERE c.post_id = $1 AND c.parent_id IS NULL
    UNION ALL
    SELECT c.*, u.handle, t.depth + 1, t.path || c.id
    FROM comments c JOIN users u ON u.id = c.author_id
    JOIN thread t ON t.id = c.parent_id
)
SELECT * FROM thread ORDER BY path, created_at ASC;
```

### Vote scoring

Votes use upsert semantics. After a vote changes, the post or comment score is recalculated from the votes table:

```sql
UPDATE posts
SET score = COALESCE(
    (SELECT SUM(direction) FROM post_votes WHERE post_id = $1), 0
)
WHERE id = $1;
```

## Migration strategy

Migrations use Goose with the `-- +goose Up` / `-- +goose Down` annotation format. Each migration file lives in `migrations/` and is named with a sequence number prefix:

```
migrations/
└── 00001_initial_schema.sql    # Full schema + seed data
```

To add a new migration:

```bash
make migrate-create NAME=add_user_karma
```

This creates a timestamped file in `migrations/`. Write the `Up` and `Down` blocks, then run:

```bash
make migrate-up
```

-- +goose Up
-- +goose StatementBegin

-- 1. SEC-DB-01: Foreign key indexes on notifications to prevent table-level lock contention and sequential scans
CREATE INDEX IF NOT EXISTS idx_notifications_post_id ON notifications (post_id);
CREATE INDEX IF NOT EXISTS idx_notifications_comment_id ON notifications (comment_id) WHERE comment_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_actor_id ON notifications (actor_id);

-- 2. SCH-04: Prevent self-notifications at the schema level
ALTER TABLE notifications ADD CONSTRAINT chk_notifications_not_self CHECK (user_id != actor_id);

-- 3. SEC-02: Rate-limiting column for post submissions
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_post_at TIMESTAMPTZ;

-- 4. SCH-06: Constrain user theme preferences to valid system themes
ALTER TABLE users ADD CONSTRAINT chk_users_theme_valid 
    CHECK (theme IN ('readit', 'catppuccin', 'nord', 'dracula', 'gruvbox', 'tokyonight'));

-- 5. SCH-05: Drop redundant B-tree index on posts superseded by idx_posts_author_keyset
DROP INDEX IF EXISTS idx_posts_author;

-- 6. PERF-05: Trigram extension and GIN index for high-performance title search
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS idx_posts_title_trgm ON posts USING gin (title gin_trgm_ops);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_posts_title_trgm;
DROP EXTENSION IF EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_posts_author ON posts (author_id, created_at DESC);

ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_theme_valid;
ALTER TABLE users DROP COLUMN IF EXISTS last_post_at;
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS chk_notifications_not_self;

DROP INDEX IF EXISTS idx_notifications_actor_id;
DROP INDEX IF EXISTS idx_notifications_comment_id;
DROP INDEX IF EXISTS idx_notifications_post_id;

-- +goose StatementEnd

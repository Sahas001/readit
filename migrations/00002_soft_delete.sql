-- +goose Up
-- +goose StatementBegin

ALTER TABLE posts
    ADD COLUMN is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN deleted_at TIMESTAMPTZ;

ALTER TABLE comments
    ADD COLUMN is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN deleted_at TIMESTAMPTZ;

-- Partial indexes for active posts (optimized for board feed listing)
CREATE INDEX idx_posts_board_active_created ON posts (board_id, created_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_posts_board_active_score   ON posts (board_id, score DESC, created_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_comments_parent_active     ON comments (parent_id) WHERE is_deleted = FALSE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_comments_parent_active;
DROP INDEX IF EXISTS idx_posts_board_active_score;
DROP INDEX IF EXISTS idx_posts_board_active_created;

ALTER TABLE comments
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS is_deleted;

ALTER TABLE posts
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS is_deleted;

-- +goose StatementEnd

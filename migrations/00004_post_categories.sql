-- +goose Up
-- +goose StatementBegin

ALTER TABLE posts ADD COLUMN IF NOT EXISTS category VARCHAR(32) NOT NULL DEFAULT 'general';
CREATE INDEX IF NOT EXISTS idx_posts_board_category ON posts (board_id, category) WHERE is_deleted = FALSE OR comment_count > 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_posts_board_category;
ALTER TABLE posts DROP COLUMN IF EXISTS category;

-- +goose StatementEnd

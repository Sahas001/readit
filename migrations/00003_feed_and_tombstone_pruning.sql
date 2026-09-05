-- +goose Up
-- +goose StatementBegin

CREATE INDEX idx_posts_board_feed_created ON posts (board_id, created_at DESC) WHERE is_deleted = FALSE OR comment_count > 0;
CREATE INDEX idx_posts_board_feed_score   ON posts (board_id, score DESC, created_at DESC) WHERE is_deleted = FALSE OR comment_count > 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_posts_board_feed_score;
DROP INDEX IF EXISTS idx_posts_board_feed_created;

-- +goose StatementEnd

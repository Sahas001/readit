-- +goose Up
-- +goose StatementBegin

-- 1. Notifications table
CREATE TABLE notifications (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id      BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id      BIGINT      NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    comment_id   BIGINT               REFERENCES comments(id) ON DELETE SET NULL,
    type         TEXT        NOT NULL CHECK (type IN ('reply_post', 'reply_comment')),
    is_read      BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Keyset index for paginated inbox browsing
CREATE INDEX idx_notifications_user_keyset 
ON notifications (user_id, created_at DESC, id DESC);

-- Partial index for instantaneous O(1) unread badge counts
CREATE INDEX idx_notifications_unread_count 
ON notifications (user_id) 
WHERE is_read = FALSE;

-- 2. Cached Karma and rate limit on users
ALTER TABLE users 
    ADD COLUMN post_karma INT NOT NULL DEFAULT 0,
    ADD COLUMN comment_karma INT NOT NULL DEFAULT 0,
    ADD COLUMN last_comment_at TIMESTAMPTZ;

-- Backfill initial karma from existing posts and comments
UPDATE users u
SET post_karma = COALESCE((
    SELECT SUM(p.score) FROM posts p WHERE p.author_id = u.id AND p.is_deleted = FALSE
), 0),
comment_karma = COALESCE((
    SELECT SUM(c.score) FROM comments c WHERE c.author_id = u.id AND c.is_deleted = FALSE
), 0);

-- 3. Profile keyset indexes
CREATE INDEX IF NOT EXISTS idx_posts_author_keyset 
ON posts (author_id, created_at DESC, id DESC) 
WHERE is_deleted = FALSE OR comment_count > 0;

CREATE INDEX IF NOT EXISTS idx_comments_author_keyset 
ON comments (author_id, created_at DESC, id DESC) 
WHERE is_deleted = FALSE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_comments_author_keyset;
DROP INDEX IF EXISTS idx_posts_author_keyset;

ALTER TABLE users 
    DROP COLUMN IF EXISTS last_comment_at,
    DROP COLUMN IF EXISTS comment_karma,
    DROP COLUMN IF EXISTS post_karma;

DROP TABLE IF EXISTS notifications;
-- +goose StatementEnd

-- +goose Up
-- +goose StatementBegin

-- 1. SEC-01: Prevent keyless/empty pubkey registrations
ALTER TABLE users ADD CONSTRAINT chk_users_pubkey_not_empty CHECK (length(trim(pubkey_sha256)) >= 10);

-- 2. SCH-02: Drop redundant duplicate index on users(pubkey_sha256)
DROP INDEX IF EXISTS idx_users_pubkey;

-- 3. PERF-03: Drop 6 obsolete/redundant indexes on posts to eliminate write amplification
DROP INDEX IF EXISTS idx_posts_board_created;
DROP INDEX IF EXISTS idx_posts_board_score;
DROP INDEX IF EXISTS idx_posts_board_active_created;
DROP INDEX IF EXISTS idx_posts_board_active_score;
DROP INDEX IF EXISTS idx_posts_board_feed_created;
DROP INDEX IF EXISTS idx_posts_board_feed_score;

-- 4. SCH-01: Enforce relational hierarchy preventing cross-post comment nesting
ALTER TABLE comments ADD CONSTRAINT uq_comments_post_id_id UNIQUE (post_id, id);
ALTER TABLE comments 
    ADD CONSTRAINT fk_comments_parent_same_post 
    FOREIGN KEY (post_id, parent_id) 
    REFERENCES comments(post_id, id) 
    ON DELETE CASCADE;

-- 5. PERF-04: Composite index for recursive CTE thread traversal
CREATE INDEX IF NOT EXISTS idx_comments_post_parent ON comments (post_id, parent_id);

-- 6. PERF-02: Keyset pagination indexes supporting category filtering
CREATE INDEX IF NOT EXISTS idx_posts_keyset_cat_hot
ON posts (board_id, category, hot_score DESC, id DESC)
WHERE is_deleted = FALSE OR comment_count > 0;

CREATE INDEX IF NOT EXISTS idx_posts_keyset_cat_top
ON posts (board_id, category, score DESC, created_at DESC, id DESC)
WHERE is_deleted = FALSE OR comment_count > 0;

CREATE INDEX IF NOT EXISTS idx_posts_keyset_cat_new
ON posts (board_id, category, created_at DESC, id DESC)
WHERE is_deleted = FALSE OR comment_count > 0;

-- 7. SCH-03: Quality-of-data check constraints
ALTER TABLE posts ADD CONSTRAINT chk_posts_title_not_blank CHECK (length(trim(title)) > 0);
ALTER TABLE comments ADD CONSTRAINT chk_comments_body_not_blank CHECK (length(trim(body)) > 0);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE comments DROP CONSTRAINT IF EXISTS chk_comments_body_not_blank;
ALTER TABLE posts DROP CONSTRAINT IF EXISTS chk_posts_title_not_blank;

DROP INDEX IF EXISTS idx_posts_keyset_cat_new;
DROP INDEX IF EXISTS idx_posts_keyset_cat_top;
DROP INDEX IF EXISTS idx_posts_keyset_cat_hot;
DROP INDEX IF EXISTS idx_comments_post_parent;

ALTER TABLE comments DROP CONSTRAINT IF EXISTS fk_comments_parent_same_post;
ALTER TABLE comments DROP CONSTRAINT IF EXISTS uq_comments_post_id_id;

CREATE INDEX idx_posts_board_feed_score ON posts (board_id, score DESC, created_at DESC) WHERE is_deleted = FALSE OR comment_count > 0;
CREATE INDEX idx_posts_board_feed_created ON posts (board_id, created_at DESC) WHERE is_deleted = FALSE OR comment_count > 0;
CREATE INDEX idx_posts_board_active_score ON posts (board_id, score DESC, created_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_posts_board_active_created ON posts (board_id, created_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_posts_board_score ON posts (board_id, score DESC, created_at DESC);
CREATE INDEX idx_posts_board_created ON posts (board_id, created_at DESC);

CREATE INDEX idx_users_pubkey ON users (pubkey_sha256);
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_pubkey_not_empty;
-- +goose StatementEnd

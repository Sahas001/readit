-- +goose Up
-- +goose StatementBegin

-- 1. Add monotonic hot_score column for indexable Hot ranking
ALTER TABLE posts ADD COLUMN IF NOT EXISTS hot_score DOUBLE PRECISION NOT NULL DEFAULT 0.0;

-- 2. Immutable formula function for monotonic hot score
CREATE OR REPLACE FUNCTION calculate_hot_score(p_score INT, p_created_at TIMESTAMPTZ)
RETURNS DOUBLE PRECISION AS $$
BEGIN
    RETURN (
        SIGN(p_score) * LOG(GREATEST(1.0, ABS(p_score)::NUMERIC)) +
        (EXTRACT(EPOCH FROM p_created_at) - 1704067200.0) / 45000.0
    )::DOUBLE PRECISION;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- 3. Trigger to maintain hot_score on INSERT and score UPDATE
CREATE OR REPLACE FUNCTION trg_posts_calculate_hot_score()
RETURNS TRIGGER AS $$
BEGIN
    NEW.hot_score := calculate_hot_score(NEW.score, NEW.created_at);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_posts_hot_score ON posts;
CREATE TRIGGER trg_posts_hot_score
BEFORE INSERT OR UPDATE OF score ON posts
FOR EACH ROW
EXECUTE FUNCTION trg_posts_calculate_hot_score();

-- 4. Backfill hot_score for existing posts
UPDATE posts SET hot_score = calculate_hot_score(score, created_at);

-- 5. Composite partial indexes for high-speed keyset pagination
CREATE INDEX IF NOT EXISTS idx_posts_keyset_new
ON posts (board_id, created_at DESC, id DESC)
WHERE is_deleted = FALSE OR comment_count > 0;

CREATE INDEX IF NOT EXISTS idx_posts_keyset_top
ON posts (board_id, score DESC, created_at DESC, id DESC)
WHERE is_deleted = FALSE OR comment_count > 0;

CREATE INDEX IF NOT EXISTS idx_posts_keyset_hot
ON posts (board_id, hot_score DESC, id DESC)
WHERE is_deleted = FALSE OR comment_count > 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_posts_keyset_hot;
DROP INDEX IF EXISTS idx_posts_keyset_top;
DROP INDEX IF EXISTS idx_posts_keyset_new;
DROP TRIGGER IF EXISTS trg_posts_hot_score ON posts;
DROP FUNCTION IF EXISTS trg_posts_calculate_hot_score();
DROP FUNCTION IF EXISTS calculate_hot_score(INT, TIMESTAMPTZ);
ALTER TABLE posts DROP COLUMN IF EXISTS hot_score;

-- +goose StatementEnd

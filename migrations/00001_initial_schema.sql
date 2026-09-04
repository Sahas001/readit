-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS citext;

-- ============================================================================
-- USERS
-- ============================================================================
CREATE TABLE users (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    pubkey_sha256   TEXT        NOT NULL UNIQUE,
    handle          CITEXT      NOT NULL UNIQUE,
    bio             TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_pubkey ON users (pubkey_sha256);

-- ============================================================================
-- BOARDS
-- ============================================================================
CREATE TABLE boards (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slug        CITEXT  NOT NULL UNIQUE,
    title       TEXT    NOT NULL,
    description TEXT    NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================================
-- POSTS
-- ============================================================================
CREATE TABLE posts (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    board_id        BIGINT      NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    author_id       BIGINT      NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    title           TEXT        NOT NULL,
    body            TEXT        NOT NULL DEFAULT '',
    url             TEXT        NOT NULL DEFAULT '',
    score           INT         NOT NULL DEFAULT 0,
    comment_count   INT         NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_posts_board_created  ON posts (board_id, created_at DESC);
CREATE INDEX idx_posts_board_score    ON posts (board_id, score DESC, created_at DESC);
CREATE INDEX idx_posts_author         ON posts (author_id, created_at DESC);

-- ============================================================================
-- COMMENTS
-- ============================================================================
CREATE TABLE comments (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    post_id     BIGINT      NOT NULL REFERENCES posts(id)    ON DELETE CASCADE,
    parent_id   BIGINT               REFERENCES comments(id) ON DELETE CASCADE,
    author_id   BIGINT      NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    body        TEXT        NOT NULL,
    score       INT         NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_comments_post    ON comments (post_id, created_at ASC);
CREATE INDEX idx_comments_parent  ON comments (parent_id);
CREATE INDEX idx_comments_author  ON comments (author_id);

-- ============================================================================
-- VOTES
-- ============================================================================
CREATE TABLE post_votes (
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id    BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    direction  SMALLINT NOT NULL CHECK (direction IN (-1, 1)),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, post_id)
);

CREATE INDEX idx_post_votes_post ON post_votes (post_id);

CREATE TABLE comment_votes (
    user_id    BIGINT   NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    comment_id BIGINT   NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    direction  SMALLINT NOT NULL CHECK (direction IN (-1, 1)),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, comment_id)
);

CREATE INDEX idx_comment_votes_comment ON comment_votes (comment_id);

-- ============================================================================
-- SEED: Default boards
-- ============================================================================
INSERT INTO boards (slug, title, description) VALUES
    ('general',     'General',      'Catch-all discussion board'),
    ('ask',         'Ask ReadIT',   'Ask the community anything'),
    ('show',        'Show ReadIT',  'Show off your projects'),
    ('meta',        'Meta',         'Feedback and site discussion');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS comment_votes;
DROP TABLE IF EXISTS post_votes;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS posts;
DROP TABLE IF EXISTS boards;
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS citext;
-- +goose StatementEnd

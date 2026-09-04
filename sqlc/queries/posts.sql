-- name: ListPostsByBoardNew :many
SELECT
    p.id,
    p.board_id,
    p.author_id,
    p.title,
    p.url,
    p.score,
    p.comment_count,
    p.created_at,
    u.handle AS author_handle
FROM posts p
JOIN users u ON u.id = p.author_id
WHERE p.board_id = $1
ORDER BY p.created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListPostsByBoardTop :many
SELECT
    p.id,
    p.board_id,
    p.author_id,
    p.title,
    p.url,
    p.score,
    p.comment_count,
    p.created_at,
    u.handle AS author_handle
FROM posts p
JOIN users u ON u.id = p.author_id
WHERE p.board_id = $1
ORDER BY p.score DESC, p.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetPostByID :one
SELECT
    p.id,
    p.board_id,
    p.author_id,
    p.title,
    p.body,
    p.url,
    p.score,
    p.comment_count,
    p.created_at,
    p.updated_at,
    u.handle AS author_handle,
    b.slug   AS board_slug
FROM posts p
JOIN users  u ON u.id = p.author_id
JOIN boards b ON b.id = p.board_id
WHERE p.id = $1;

-- name: CreatePost :one
INSERT INTO posts (board_id, author_id, title, body, url)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, board_id, author_id, title, body, url, score, comment_count, created_at, updated_at;

-- name: IncrementPostCommentCount :exec
UPDATE posts SET comment_count = comment_count + 1 WHERE id = $1;

-- name: DecrementPostCommentCount :exec
UPDATE posts SET comment_count = GREATEST(comment_count - 1, 0) WHERE id = $1;

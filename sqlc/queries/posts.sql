-- name: ListPostsByBoardHot :many
SELECT
    p.id,
    p.board_id,
    p.author_id,
    p.title,
    p.url,
    p.score,
    p.comment_count,
    p.created_at,
    p.is_deleted,
    p.category,
    (CASE WHEN p.is_deleted THEN '[deleted]' ELSE u.handle END)::TEXT AS author_handle
FROM posts p
JOIN users u ON u.id = p.author_id
WHERE p.board_id = $1
  AND (p.is_deleted = FALSE OR p.comment_count > 0)
  AND (sqlc.arg(category)::TEXT = '' OR p.category = sqlc.arg(category))
ORDER BY (
    (p.score + 1)::FLOAT / POWER(GREATEST(1.0, EXTRACT(EPOCH FROM (now() - p.created_at))/3600.0 + 2.0), 1.5)
) DESC, p.created_at DESC
LIMIT $2 OFFSET $3;

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
    p.is_deleted,
    p.category,
    (CASE WHEN p.is_deleted THEN '[deleted]' ELSE u.handle END)::TEXT AS author_handle
FROM posts p
JOIN users u ON u.id = p.author_id
WHERE p.board_id = $1
  AND (p.is_deleted = FALSE OR p.comment_count > 0)
  AND (sqlc.arg(category)::TEXT = '' OR p.category = sqlc.arg(category))
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
    p.is_deleted,
    p.category,
    (CASE WHEN p.is_deleted THEN '[deleted]' ELSE u.handle END)::TEXT AS author_handle
FROM posts p
JOIN users u ON u.id = p.author_id
WHERE p.board_id = $1
  AND (p.is_deleted = FALSE OR p.comment_count > 0)
  AND (sqlc.arg(category)::TEXT = '' OR p.category = sqlc.arg(category))
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
    p.is_deleted,
    p.category,
    (CASE WHEN p.is_deleted THEN '[deleted]' ELSE u.handle END)::TEXT AS author_handle,
    b.slug   AS board_slug
FROM posts p
JOIN users  u ON u.id = p.author_id
JOIN boards b ON b.id = p.board_id
WHERE p.id = $1;

-- name: CreatePost :one
INSERT INTO posts (board_id, author_id, title, body, url, category)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: IncrementPostCommentCount :exec
UPDATE posts SET comment_count = comment_count + 1 WHERE id = $1;

-- name: DecrementPostCommentCount :exec
UPDATE posts SET comment_count = GREATEST(comment_count - 1, 0) WHERE id = $1;

-- name: RecalculatePostCommentCount :exec
UPDATE posts
SET comment_count = (
    SELECT COUNT(*)::INT FROM comments WHERE post_id = $1 AND is_deleted = FALSE
)
WHERE id = $1;

-- name: HasPostComments :one
SELECT EXISTS(
    SELECT 1 FROM comments WHERE post_id = $1 AND is_deleted = FALSE
)::BOOLEAN;

-- name: HardDeletePost :exec
DELETE FROM posts
WHERE id = $1 AND author_id = $2;

-- name: SoftDeletePost :exec
UPDATE posts
SET is_deleted = TRUE,
    deleted_at = now(),
    title = '[deleted]',
    body = '',
    url = ''
WHERE id = $1 AND author_id = $2;

-- name: PruneDeletedPostIfEmpty :exec
DELETE FROM posts
WHERE posts.id = $1
  AND posts.is_deleted = TRUE
  AND (
      posts.comment_count = 0
      OR NOT EXISTS (
          SELECT 1 FROM comments WHERE comments.post_id = posts.id AND comments.is_deleted = FALSE
      )
  );

-- name: PruneAllEmptyDeletedPosts :exec
DELETE FROM posts
WHERE is_deleted = TRUE
  AND (
      comment_count = 0
      OR NOT EXISTS (
          SELECT 1 FROM comments WHERE comments.post_id = posts.id AND comments.is_deleted = FALSE
      )
  );


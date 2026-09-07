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
    p.hot_score,
    (CASE WHEN p.is_deleted THEN '[deleted]' ELSE u.handle END)::TEXT AS author_handle
FROM posts p
JOIN users u ON u.id = p.author_id
WHERE p.board_id = $1
  AND (p.is_deleted = FALSE OR p.comment_count > 0)
  AND (sqlc.arg(category)::TEXT = '' OR p.category = sqlc.arg(category))
  AND (sqlc.arg(search_query)::TEXT = '' OR (p.title ILIKE '%' || sqlc.arg(search_query)::TEXT || '%' OR p.body ILIKE '%' || sqlc.arg(search_query)::TEXT || '%'))
  AND (
      sqlc.narg(cursor_hot_score)::FLOAT8 IS NULL
      OR (p.hot_score < sqlc.narg(cursor_hot_score)::FLOAT8)
      OR (p.hot_score = sqlc.narg(cursor_hot_score)::FLOAT8 AND p.id < sqlc.narg(cursor_id)::BIGINT)
  )
ORDER BY p.hot_score DESC, p.id DESC
LIMIT $2;

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
    p.hot_score,
    (CASE WHEN p.is_deleted THEN '[deleted]' ELSE u.handle END)::TEXT AS author_handle
FROM posts p
JOIN users u ON u.id = p.author_id
WHERE p.board_id = $1
  AND (p.is_deleted = FALSE OR p.comment_count > 0)
  AND (sqlc.arg(category)::TEXT = '' OR p.category = sqlc.arg(category))
  AND (sqlc.arg(search_query)::TEXT = '' OR (p.title ILIKE '%' || sqlc.arg(search_query)::TEXT || '%' OR p.body ILIKE '%' || sqlc.arg(search_query)::TEXT || '%'))
  AND (
      sqlc.narg(cursor_created_at)::TIMESTAMPTZ IS NULL
      OR (p.created_at < sqlc.narg(cursor_created_at)::TIMESTAMPTZ)
      OR (p.created_at = sqlc.narg(cursor_created_at)::TIMESTAMPTZ AND p.id < sqlc.narg(cursor_id)::BIGINT)
  )
ORDER BY p.created_at DESC, p.id DESC
LIMIT $2;

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
    p.hot_score,
    (CASE WHEN p.is_deleted THEN '[deleted]' ELSE u.handle END)::TEXT AS author_handle
FROM posts p
JOIN users u ON u.id = p.author_id
WHERE p.board_id = $1
  AND (p.is_deleted = FALSE OR p.comment_count > 0)
  AND (sqlc.arg(category)::TEXT = '' OR p.category = sqlc.arg(category))
  AND (sqlc.arg(search_query)::TEXT = '' OR (p.title ILIKE '%' || sqlc.arg(search_query)::TEXT || '%' OR p.body ILIKE '%' || sqlc.arg(search_query)::TEXT || '%'))
  AND (
      sqlc.narg(cursor_score)::INT IS NULL
      OR (p.score < sqlc.narg(cursor_score)::INT)
      OR (p.score = sqlc.narg(cursor_score)::INT AND (
          (p.created_at < sqlc.narg(cursor_created_at)::TIMESTAMPTZ)
          OR (p.created_at = sqlc.narg(cursor_created_at)::TIMESTAMPTZ AND p.id < sqlc.narg(cursor_id)::BIGINT)
      ))
  )
ORDER BY p.score DESC, p.created_at DESC, p.id DESC
LIMIT $2;

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

-- name: HardDeletePostIfEmpty :execrows
DELETE FROM posts
WHERE posts.id = $1 AND posts.author_id = $2
  AND NOT EXISTS (
      SELECT 1 FROM comments WHERE comments.post_id = posts.id
  );

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


-- name: GetCommentThreadByPost :many
-- Recursive CTE that returns the comment tree for a post up to depth 15,
-- ordered depth-first by (path, created_at).
WITH RECURSIVE thread AS (
    -- Anchor: top-level comments (parent_id IS NULL)
    SELECT
        c.id,
        c.post_id,
        c.parent_id,
        c.author_id,
        c.body,
        c.score,
        c.created_at,
        c.updated_at,
        c.is_deleted,
        (CASE WHEN c.is_deleted THEN '[deleted]' ELSE u.handle END)::TEXT AS author_handle,
        0::INT   AS depth,
        ARRAY[c.id] AS path
    FROM comments c
    JOIN users u ON u.id = c.author_id
    WHERE c.post_id = $1 AND c.parent_id IS NULL

    UNION ALL

    -- Recursive: child comments (strictly constrained to same post_id with depth ceiling)
    SELECT
        c.id,
        c.post_id,
        c.parent_id,
        c.author_id,
        c.body,
        c.score,
        c.created_at,
        c.updated_at,
        c.is_deleted,
        (CASE WHEN c.is_deleted THEN '[deleted]' ELSE u.handle END)::TEXT AS author_handle,
        t.depth + 1,
        t.path || c.id
    FROM comments c
    JOIN users u ON u.id = c.author_id
    JOIN thread t ON t.id = c.parent_id
    WHERE c.post_id = $1 AND t.depth < 15
)
SELECT id, post_id, parent_id, author_id, body, score,
       created_at, updated_at, is_deleted, author_handle, depth, path
FROM thread
ORDER BY path, created_at ASC
LIMIT $2;

-- name: CreateComment :one
INSERT INTO comments (post_id, parent_id, author_id, body)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: HasCommentChildren :one
SELECT EXISTS(
    SELECT 1 FROM comments WHERE parent_id = $1
)::BOOLEAN;

-- name: HardDeleteComment :exec
DELETE FROM comments
WHERE id = $1 AND author_id = $2;

-- name: HardDeleteCommentIfNoChildren :execrows
DELETE FROM comments
WHERE comments.id = $1 AND comments.author_id = $2
  AND NOT EXISTS (
      SELECT 1 FROM comments c WHERE c.parent_id = comments.id
  );

-- name: SoftDeleteComment :exec
UPDATE comments
SET is_deleted = TRUE,
    deleted_at = now(),
    body = '[deleted]'
WHERE id = $1 AND author_id = $2;

-- name: PruneTombstoneComments :exec
WITH RECURSIVE active_ancestors AS (
    SELECT parent_id, 1 AS depth
    FROM comments
    WHERE post_id = $1 AND is_deleted = FALSE AND parent_id IS NOT NULL
    UNION
    SELECT c.parent_id, a.depth + 1
    FROM comments c
    JOIN active_ancestors a ON c.id = a.parent_id
    WHERE c.post_id = $1 AND c.parent_id IS NOT NULL AND a.depth < 15
)
DELETE FROM comments
WHERE comments.post_id = $1
  AND comments.is_deleted = TRUE
  AND comments.id NOT IN (SELECT parent_id FROM active_ancestors);

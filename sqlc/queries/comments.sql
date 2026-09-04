-- name: GetCommentThreadByPost :many
-- Recursive CTE that returns the full comment tree for a post,
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
        u.handle AS author_handle,
        0::INT   AS depth,
        ARRAY[c.id] AS path
    FROM comments c
    JOIN users u ON u.id = c.author_id
    WHERE c.post_id = $1 AND c.parent_id IS NULL

    UNION ALL

    -- Recursive: child comments
    SELECT
        c.id,
        c.post_id,
        c.parent_id,
        c.author_id,
        c.body,
        c.score,
        c.created_at,
        c.updated_at,
        u.handle AS author_handle,
        t.depth + 1,
        t.path || c.id
    FROM comments c
    JOIN users u ON u.id = c.author_id
    JOIN thread t ON t.id = c.parent_id
)
SELECT id, post_id, parent_id, author_id, body, score,
       created_at, updated_at, author_handle, depth, path
FROM thread
ORDER BY path, created_at ASC;

-- name: CreateComment :one
INSERT INTO comments (post_id, parent_id, author_id, body)
VALUES ($1, $2, $3, $4)
RETURNING id, post_id, parent_id, author_id, body, score, created_at, updated_at;

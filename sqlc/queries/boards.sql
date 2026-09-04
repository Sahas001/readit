-- name: ListBoards :many
SELECT id, slug, title, description, created_at
FROM boards
ORDER BY slug ASC;

-- name: GetBoardBySlug :one
SELECT id, slug, title, description, created_at
FROM boards
WHERE slug = $1;

-- name: GetBoardByID :one
SELECT id, slug, title, description, created_at
FROM boards
WHERE id = $1;

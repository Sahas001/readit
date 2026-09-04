-- name: GetUserByPubkey :one
SELECT id, pubkey_sha256, handle, bio, created_at, updated_at
FROM users
WHERE pubkey_sha256 = $1;

-- name: UpsertUser :one
INSERT INTO users (pubkey_sha256, handle)
VALUES ($1, $2)
ON CONFLICT (pubkey_sha256)
DO UPDATE SET updated_at = now()
RETURNING id, pubkey_sha256, handle, bio, created_at, updated_at;

-- name: UpdateUserHandle :one
UPDATE users
SET handle = $2, updated_at = now()
WHERE id = $1
RETURNING id, pubkey_sha256, handle, bio, created_at, updated_at;

-- name: UpdateUserBio :one
UPDATE users
SET bio = $2, updated_at = now()
WHERE id = $1
RETURNING id, pubkey_sha256, handle, bio, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, pubkey_sha256, handle, bio, created_at, updated_at
FROM users
WHERE id = $1;

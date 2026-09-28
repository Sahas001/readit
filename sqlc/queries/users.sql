-- name: GetUserByPubkey :one
SELECT id, pubkey_sha256, handle, bio, created_at, updated_at, post_karma, comment_karma, last_comment_at, theme
FROM users
WHERE pubkey_sha256 = $1;

-- name: UpsertUser :one
INSERT INTO users (pubkey_sha256, handle)
VALUES ($1, $2)
ON CONFLICT (pubkey_sha256)
DO UPDATE SET updated_at = now()
RETURNING id, pubkey_sha256, handle, bio, created_at, updated_at, post_karma, comment_karma, last_comment_at, theme;

-- name: UpdateUserHandle :one
UPDATE users
SET handle = $2, updated_at = now()
WHERE id = $1
RETURNING id, pubkey_sha256, handle, bio, created_at, updated_at, post_karma, comment_karma, last_comment_at, theme;

-- name: UpdateUserBio :one
UPDATE users
SET bio = $2, updated_at = now()
WHERE id = $1
RETURNING id, pubkey_sha256, handle, bio, created_at, updated_at, post_karma, comment_karma, last_comment_at, theme;

-- name: GetUserByID :one
SELECT id, pubkey_sha256, handle, bio, created_at, updated_at, post_karma, comment_karma, last_comment_at, theme
FROM users
WHERE id = $1;

-- name: GetUserProfileByHandle :one
SELECT id, pubkey_sha256, handle, bio, created_at, updated_at, post_karma, comment_karma, last_comment_at, theme
FROM users
WHERE handle = $1;

-- name: AdjustUserPostKarma :exec
UPDATE users
SET post_karma = post_karma + $2, updated_at = now()
WHERE id = $1;

-- name: AdjustUserCommentKarma :exec
UPDATE users
SET comment_karma = comment_karma + $2, updated_at = now()
WHERE id = $1;

-- name: UpdateUserLastCommentAt :exec
UPDATE users
SET last_comment_at = now()
WHERE id = $1;

-- name: UpdateUserTheme :one
UPDATE users
SET theme = $2, updated_at = now()
WHERE id = $1
RETURNING id, pubkey_sha256, handle, bio, created_at, updated_at, post_karma, comment_karma, last_comment_at, theme;

-- name: CreateNotification :exec
INSERT INTO notifications (user_id, actor_id, post_id, comment_id, type)
VALUES ($1, $2, $3, $4, $5);

-- name: GetUnreadNotificationCount :one
SELECT COUNT(*)::BIGINT
FROM notifications
WHERE user_id = $1 AND is_read = FALSE;

-- name: ListNotificationsKeyset :many
SELECT n.id, n.user_id, n.actor_id, n.post_id, n.comment_id, n.type, n.is_read, n.created_at,
       u.handle AS actor_handle,
       p.title  AS post_title,
       COALESCE(c.body, '[deleted]') AS comment_snippet
FROM notifications n
JOIN users u ON u.id = n.actor_id
JOIN posts p ON p.id = n.post_id
LEFT JOIN comments c ON c.id = n.comment_id
WHERE n.user_id = $1
  AND (
      sqlc.narg(cursor_created_at)::TIMESTAMPTZ IS NULL
      OR (n.created_at < sqlc.narg(cursor_created_at)::TIMESTAMPTZ)
      OR (n.created_at = sqlc.narg(cursor_created_at)::TIMESTAMPTZ AND n.id < sqlc.narg(cursor_id)::BIGINT)
  )
ORDER BY n.created_at DESC, n.id DESC
LIMIT $2;

-- name: MarkNotificationsRead :exec
UPDATE notifications
SET is_read = TRUE
WHERE user_id = $1 AND is_read = FALSE;

-- name: MarkNotificationAsReadByID :exec
UPDATE notifications
SET is_read = TRUE
WHERE id = $1 AND user_id = $2;

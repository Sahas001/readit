-- name: UpsertPostVote :exec
INSERT INTO post_votes (user_id, post_id, direction)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, post_id)
DO UPDATE SET direction = EXCLUDED.direction, created_at = now();

-- name: DeletePostVote :exec
DELETE FROM post_votes WHERE user_id = $1 AND post_id = $2;

-- name: GetPostVoteByUser :one
SELECT user_id, post_id, direction, created_at
FROM post_votes
WHERE user_id = $1 AND post_id = $2;

-- name: RecalculatePostScore :exec
UPDATE posts
SET score = COALESCE((SELECT SUM(direction) FROM post_votes WHERE post_id = $1), 0)
WHERE id = $1;

-- name: UpsertCommentVote :exec
INSERT INTO comment_votes (user_id, comment_id, direction)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, comment_id)
DO UPDATE SET direction = EXCLUDED.direction, created_at = now();

-- name: DeleteCommentVote :exec
DELETE FROM comment_votes WHERE user_id = $1 AND comment_id = $2;

-- name: GetCommentVoteByUser :one
SELECT user_id, comment_id, direction, created_at
FROM comment_votes
WHERE user_id = $1 AND comment_id = $2;

-- name: RecalculateCommentScore :exec
UPDATE comments
SET score = COALESCE((SELECT SUM(direction) FROM comment_votes WHERE comment_id = $1), 0)
WHERE id = $1;

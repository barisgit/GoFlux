-- name: GetCommentsByPostID :many
SELECT c.id,
    c.content,
    c.post_id,
    c.user_id,
    c.created_at,
    c.updated_at,
    u.name as user_name,
    u.email as user_email
FROM comments c
    JOIN users u ON c.user_id = u.id
WHERE c.post_id = $1
ORDER BY c.created_at ASC;
-- name: GetCommentByID :one
SELECT c.id,
    c.content,
    c.post_id,
    c.user_id,
    c.created_at,
    c.updated_at,
    u.name as user_name,
    u.email as user_email
FROM comments c
    JOIN users u ON c.user_id = u.id
WHERE c.id = $1;
-- name: CreateComment :one
INSERT INTO comments (content, post_id, user_id)
VALUES ($1, $2, $3)
RETURNING id,
    content,
    post_id,
    user_id,
    created_at,
    updated_at;
-- name: UpdateComment :one
UPDATE comments
SET content = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING id,
    content,
    post_id,
    user_id,
    created_at,
    updated_at;
-- name: DeleteComment :exec
DELETE FROM comments
WHERE id = $1;
-- name: CountCommentsByPost :one
SELECT COUNT(*)
FROM comments
WHERE post_id = $1;
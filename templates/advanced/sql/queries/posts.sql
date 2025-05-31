-- name: GetPosts :many
SELECT p.id,
    p.title,
    p.content,
    p.user_id,
    p.published,
    p.created_at,
    p.updated_at,
    u.name as user_name,
    u.email as user_email
FROM posts p
    JOIN users u ON p.user_id = u.id
ORDER BY p.created_at DESC;
-- name: GetPostByID :one
SELECT p.id,
    p.title,
    p.content,
    p.user_id,
    p.published,
    p.created_at,
    p.updated_at,
    u.name as user_name,
    u.email as user_email
FROM posts p
    JOIN users u ON p.user_id = u.id
WHERE p.id = $1;
-- name: GetPostsByUserID :many
SELECT id,
    title,
    content,
    user_id,
    published,
    created_at,
    updated_at
FROM posts
WHERE user_id = $1
ORDER BY created_at DESC;
-- name: GetPublishedPosts :many
SELECT p.id,
    p.title,
    p.content,
    p.user_id,
    p.published,
    p.created_at,
    p.updated_at,
    u.name as user_name,
    u.email as user_email
FROM posts p
    JOIN users u ON p.user_id = u.id
WHERE p.published = true
ORDER BY p.created_at DESC;
-- name: CreatePost :one
INSERT INTO posts (title, content, user_id, published)
VALUES ($1, $2, $3, $4)
RETURNING id,
    title,
    content,
    user_id,
    published,
    created_at,
    updated_at;
-- name: UpdatePost :one
UPDATE posts
SET title = $2,
    content = $3,
    user_id = $4,
    published = $5,
    updated_at = NOW()
WHERE id = $1
RETURNING id,
    title,
    content,
    user_id,
    published,
    created_at,
    updated_at;
-- name: UpdatePostPublished :one
UPDATE posts
SET published = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING id,
    title,
    content,
    user_id,
    published,
    created_at,
    updated_at;
-- name: DeletePost :exec
DELETE FROM posts
WHERE id = $1;
-- name: CountPosts :one
SELECT COUNT(*)
FROM posts;
-- name: CountPostsByUser :one
SELECT COUNT(*)
FROM posts
WHERE user_id = $1;
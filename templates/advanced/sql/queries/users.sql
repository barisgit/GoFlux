-- name: GetUsers :many
SELECT id,
    name,
    email,
    age,
    created_at,
    updated_at
FROM users
ORDER BY created_at DESC;
-- name: GetUserByID :one
SELECT id,
    name,
    email,
    age,
    created_at,
    updated_at
FROM users
WHERE id = $1;
-- name: GetUserByEmail :one
SELECT id,
    name,
    email,
    age,
    created_at,
    updated_at
FROM users
WHERE email = $1;
-- name: CreateUser :one
INSERT INTO users (name, email, age)
VALUES ($1, $2, $3)
RETURNING id,
    name,
    email,
    age,
    created_at,
    updated_at;
-- name: UpdateUser :one
UPDATE users
SET name = $2,
    email = $3,
    age = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING id,
    name,
    email,
    age,
    created_at,
    updated_at;
-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;
-- name: CountUsers :one
SELECT COUNT(*)
FROM users;
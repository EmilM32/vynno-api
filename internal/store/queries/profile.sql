-- name: GetProfile :one
SELECT display_name, handle, avatar_url
FROM profiles
WHERE user_id = $1;

-- name: CountUsers :one
SELECT count(*)::bigint FROM users;

-- name: InsertUser :exec
INSERT INTO users (id, username, password_hash) VALUES ($1, $2, $3);

-- name: GetUserByUsername :one
SELECT id, username, password_hash
FROM users
WHERE username = $1;

-- name: GetUserByID :one
SELECT id, username, password_hash
FROM users
WHERE id = $1;

-- name: SetUserCredentials :exec
UPDATE users
SET username = $2, password_hash = $3
WHERE id = $1;

-- name: UsernameInUse :one
SELECT EXISTS(
    SELECT 1 FROM users WHERE username = $1 AND id <> $2
)::boolean;

-- name: InsertProfile :exec
INSERT INTO profiles (user_id, display_name, handle, avatar_url)
VALUES ($1, $2, $3, $4);

-- name: FirstUserID :one
SELECT id FROM users ORDER BY id LIMIT 1;

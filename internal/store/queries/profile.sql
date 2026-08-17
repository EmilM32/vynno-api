-- name: GetProfile :one
SELECT display_name, handle, avatar_url
FROM profiles
WHERE user_id = $1;

-- name: CountUsers :one
SELECT count(*)::bigint FROM users;

-- name: InsertUser :exec
INSERT INTO users (id) VALUES ($1);

-- name: InsertProfile :exec
INSERT INTO profiles (user_id, display_name, handle, avatar_url)
VALUES ($1, $2, $3, $4);

-- name: FirstUserID :one
SELECT id FROM users ORDER BY id LIMIT 1;

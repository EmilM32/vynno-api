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

-- name: UpdateProfileDisplayName :one
UPDATE profiles
SET display_name = $2
WHERE user_id = $1
RETURNING display_name, handle, avatar_url;

-- name: SetProfileAvatarURL :exec
UPDATE profiles
SET avatar_url = $2
WHERE user_id = $1;

-- name: InsertAvatar :exec
INSERT INTO avatars (id, user_id, content_type, bytes)
VALUES ($1, $2, $3, $4);

-- name: DeleteAvatarByUser :exec
DELETE FROM avatars
WHERE user_id = $1;

-- name: GetAvatar :one
SELECT id, user_id, content_type, bytes
FROM avatars
WHERE id = $1;

-- name: FirstUserID :one
SELECT id FROM users ORDER BY id LIMIT 1;

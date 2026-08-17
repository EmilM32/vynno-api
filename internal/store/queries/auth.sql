-- name: InsertAuthToken :exec
INSERT INTO auth_tokens (id, user_id, token_hash, expires_at)
VALUES ($1, $2, $3, $4);

-- name: GetAuthTokenByHash :one
SELECT id, user_id, token_hash, expires_at
FROM auth_tokens
WHERE token_hash = $1;

-- name: DeleteAuthTokenByHash :exec
DELETE FROM auth_tokens WHERE token_hash = $1;

-- name: InsertAuthToken :exec
INSERT INTO auth_tokens (id, user_id, token_hash, expires_at, scope)
VALUES ($1, $2, $3, $4, $5);

-- name: GetAuthTokenByHash :one
SELECT id, user_id, token_hash, expires_at, scope
FROM auth_tokens
WHERE token_hash = $1;

-- name: DeleteAuthTokenByHash :exec
DELETE FROM auth_tokens WHERE token_hash = $1;

-- name: DeleteAuthTokensByUser :exec
DELETE FROM auth_tokens WHERE user_id = $1;

-- name: DeleteOtherAuthTokens :exec
DELETE FROM auth_tokens WHERE user_id = $1 AND token_hash <> $2;

-- name: DeleteExpiredAuthTokens :execrows
DELETE FROM auth_tokens WHERE expires_at <= $1;

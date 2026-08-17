-- +goose Up

ALTER TABLE users
    ADD COLUMN username TEXT UNIQUE,
    ADD COLUMN password_hash TEXT;

CREATE TABLE auth_tokens (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX auth_tokens_user_idx ON auth_tokens (user_id);

-- +goose Down

DROP INDEX IF EXISTS auth_tokens_user_idx;
DROP TABLE IF EXISTS auth_tokens;
ALTER TABLE users
    DROP COLUMN IF EXISTS password_hash,
    DROP COLUMN IF EXISTS username;

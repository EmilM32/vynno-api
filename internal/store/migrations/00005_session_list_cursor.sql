-- +goose Up

DROP INDEX IF EXISTS sessions_user_started_idx;

CREATE INDEX sessions_user_started_id_idx
    ON sessions (user_id, started_at DESC, id DESC);

-- +goose Down

DROP INDEX IF EXISTS sessions_user_started_id_idx;

CREATE INDEX sessions_user_started_idx
    ON sessions (user_id, started_at DESC);

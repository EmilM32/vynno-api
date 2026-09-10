-- +goose Up

-- A prior local 00009 added session_pauses; the product no longer pauses.
DROP TABLE IF EXISTS session_pauses;

UPDATE sessions
SET status = 'stopped',
    ended_at = paused_at,
    paused_at = NULL
WHERE status = 'paused' AND paused_at IS NOT NULL;

UPDATE sessions
SET status = 'stopped',
    ended_at = COALESCE(ended_at, started_at + interval '1 millisecond'),
    paused_at = NULL
WHERE status = 'paused';

ALTER TABLE sessions DROP COLUMN paused_at;
ALTER TABLE sessions DROP COLUMN paused_ms;

ALTER TABLE sessions DROP CONSTRAINT sessions_status_check;
ALTER TABLE sessions ADD CONSTRAINT sessions_status_check
    CHECK (status IN ('active', 'stopped'));

DROP INDEX sessions_one_live_per_user;
CREATE UNIQUE INDEX sessions_one_live_per_user
    ON sessions (user_id)
    WHERE status = 'active';

-- +goose Down

DROP INDEX sessions_one_live_per_user;
CREATE UNIQUE INDEX sessions_one_live_per_user
    ON sessions (user_id)
    WHERE status IN ('active', 'paused');

ALTER TABLE sessions DROP CONSTRAINT sessions_status_check;
ALTER TABLE sessions ADD CONSTRAINT sessions_status_check
    CHECK (status IN ('active', 'paused', 'stopped'));

ALTER TABLE sessions ADD COLUMN paused_ms BIGINT NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN paused_at TIMESTAMPTZ;

-- +goose Up

-- The per-session target left the product (EMI-147, EMI-152); the daily target lives in user_prefs.
ALTER TABLE sessions DROP COLUMN target_duration_ms;

-- +goose Down

ALTER TABLE sessions ADD COLUMN target_duration_ms BIGINT;

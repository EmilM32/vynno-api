-- +goose Up

ALTER TABLE sessions DROP COLUMN tags;

-- +goose Down

ALTER TABLE sessions ADD COLUMN tags JSONB NOT NULL DEFAULT '[]'::jsonb;

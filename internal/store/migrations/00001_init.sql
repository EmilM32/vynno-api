-- +goose Up

CREATE TABLE users (
    id UUID PRIMARY KEY
);

CREATE TABLE profiles (
    user_id UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    display_name TEXT NOT NULL,
    handle TEXT NOT NULL,
    avatar_url TEXT
);

CREATE TABLE projects (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    color TEXT NOT NULL,
    code TEXT,
    progress_percent INTEGER,
    archived BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX projects_user_code_uidx
    ON projects (user_id, lower(code))
    WHERE code IS NOT NULL;

CREATE TABLE sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects (id),
    note TEXT NOT NULL,
    ticket_id TEXT,
    activity_type TEXT,
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    status TEXT NOT NULL CHECK (status IN ('active', 'paused', 'stopped')),
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ,
    paused_ms BIGINT NOT NULL DEFAULT 0,
    paused_at TIMESTAMPTZ,
    target_duration_ms BIGINT
);

CREATE UNIQUE INDEX sessions_one_live_per_user
    ON sessions (user_id)
    WHERE status IN ('active', 'paused');

CREATE INDEX sessions_user_started_idx
    ON sessions (user_id, started_at DESC);

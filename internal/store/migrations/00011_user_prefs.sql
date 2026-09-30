-- +goose Up

-- Account-wide preferences. No row means every pref is unset.
CREATE TABLE user_prefs (
    user_id UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    daily_target_ms BIGINT
        CONSTRAINT user_prefs_daily_target_check
        CHECK (daily_target_ms BETWEEN 60000 AND 86400000),
    -- Hard-deleting the project clears the default instead of blocking the delete.
    default_project_id UUID REFERENCES projects (id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down

DROP TABLE IF EXISTS user_prefs;

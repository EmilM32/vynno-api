-- +goose Up

CREATE TABLE activity_types (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    color TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX activity_types_user_name_uidx
    ON activity_types (user_id, lower(name));

ALTER TABLE sessions
    ADD COLUMN activity_type_id UUID REFERENCES activity_types (id);

INSERT INTO activity_types (id, user_id, name, color)
SELECT gen_random_uuid(), s.user_id, s.activity_type,
    CASE s.activity_type
        WHEN 'meeting' THEN 'tertiary'
        WHEN 'coding' THEN 'secondary'
        WHEN 'debugging' THEN 'error'
        WHEN 'docs' THEN 'on-surface-variant'
        WHEN 'other' THEN 'outline'
        ELSE 'primary'
    END
FROM (
    SELECT DISTINCT ON (user_id, lower(activity_type))
        user_id, activity_type
    FROM sessions
    WHERE activity_type IS NOT NULL AND btrim(activity_type) <> ''
    ORDER BY user_id, lower(activity_type), activity_type
) s;

UPDATE sessions sess
SET activity_type_id = at.id
FROM activity_types at
WHERE at.user_id = sess.user_id
  AND at.name = sess.activity_type;

ALTER TABLE sessions DROP COLUMN activity_type;

-- +goose Down

ALTER TABLE sessions ADD COLUMN activity_type TEXT;

UPDATE sessions sess
SET activity_type = at.name
FROM activity_types at
WHERE at.id = sess.activity_type_id;

ALTER TABLE sessions DROP COLUMN activity_type_id;
DROP INDEX IF EXISTS activity_types_user_name_uidx;
DROP TABLE IF EXISTS activity_types;

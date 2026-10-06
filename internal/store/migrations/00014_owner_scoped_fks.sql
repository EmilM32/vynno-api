-- +goose Up

-- A session, and the default project in user_prefs, may only point at rows of the
-- same user. The service already checks ownership; these keys make the database
-- refuse a cross-account reference too. (user_id, id) is unique because id is.
ALTER TABLE projects ADD CONSTRAINT projects_user_id_id_key UNIQUE (user_id, id);
ALTER TABLE activity_types ADD CONSTRAINT activity_types_user_id_id_key UNIQUE (user_id, id);

ALTER TABLE sessions DROP CONSTRAINT sessions_project_id_fkey;
ALTER TABLE sessions ADD CONSTRAINT sessions_project_id_fkey
    FOREIGN KEY (user_id, project_id) REFERENCES projects (user_id, id);

-- A NULL activity_type_id skips the check (MATCH SIMPLE), as before.
ALTER TABLE sessions DROP CONSTRAINT sessions_activity_type_id_fkey;
ALTER TABLE sessions ADD CONSTRAINT sessions_activity_type_id_fkey
    FOREIGN KEY (user_id, activity_type_id) REFERENCES activity_types (user_id, id);

-- Deleting the project clears only default_project_id, never user_id (PostgreSQL 15+).
ALTER TABLE user_prefs DROP CONSTRAINT user_prefs_default_project_id_fkey;
ALTER TABLE user_prefs ADD CONSTRAINT user_prefs_default_project_id_fkey
    FOREIGN KEY (user_id, default_project_id) REFERENCES projects (user_id, id)
    ON DELETE SET NULL (default_project_id);

-- +goose Down

ALTER TABLE user_prefs DROP CONSTRAINT user_prefs_default_project_id_fkey;
ALTER TABLE user_prefs ADD CONSTRAINT user_prefs_default_project_id_fkey
    FOREIGN KEY (default_project_id) REFERENCES projects (id) ON DELETE SET NULL;

ALTER TABLE sessions DROP CONSTRAINT sessions_activity_type_id_fkey;
ALTER TABLE sessions ADD CONSTRAINT sessions_activity_type_id_fkey
    FOREIGN KEY (activity_type_id) REFERENCES activity_types (id);

ALTER TABLE sessions DROP CONSTRAINT sessions_project_id_fkey;
ALTER TABLE sessions ADD CONSTRAINT sessions_project_id_fkey
    FOREIGN KEY (project_id) REFERENCES projects (id);

ALTER TABLE activity_types DROP CONSTRAINT activity_types_user_id_id_key;
ALTER TABLE projects DROP CONSTRAINT projects_user_id_id_key;

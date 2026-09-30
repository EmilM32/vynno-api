-- name: GetPrefs :one
SELECT daily_target_ms, default_project_id
FROM user_prefs
WHERE user_id = $1;

-- name: UpsertPrefs :exec
INSERT INTO user_prefs (user_id, daily_target_ms, default_project_id, updated_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (user_id) DO UPDATE
SET daily_target_ms = EXCLUDED.daily_target_ms,
    default_project_id = EXCLUDED.default_project_id,
    updated_at = now();

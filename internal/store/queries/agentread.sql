-- Read models for the local MCP. Every statement is scoped by user_id.
-- Do not select password_hash, token_hash, code_hash, or avatar bytes.

-- name: ListAgentProjects :many
SELECT id, name, color, code, progress_percent, archived
FROM projects
WHERE user_id = $1
  AND (archived = FALSE OR sqlc.arg(include_archived)::boolean = TRUE)
ORDER BY name ASC;

-- name: ListAgentSessions :many
SELECT
    s.id,
    s.project_id,
    p.name AS project_name,
    s.note,
    s.ticket_id,
    s.activity_type_id,
    COALESCE(a.name, '') AS activity_type_name,
    s.status,
    s.started_at,
    s.ended_at,
    s.target_duration_ms
FROM sessions s
JOIN projects p ON p.id = s.project_id AND p.user_id = s.user_id
LEFT JOIN activity_types a ON a.id = s.activity_type_id AND a.user_id = s.user_id
WHERE s.user_id = sqlc.arg(user_id)
  AND (
    sqlc.arg(filter_project)::boolean = FALSE
    OR s.project_id = sqlc.arg(project_id)::uuid
  )
  AND (
    sqlc.arg(filter_status)::boolean = FALSE
    OR s.status = sqlc.arg(status)
  )
  AND (
    sqlc.arg(filter_from)::boolean = FALSE
    OR s.ended_at IS NULL
    OR s.ended_at > sqlc.arg(from_ts)::timestamptz
  )
  AND (
    sqlc.arg(filter_to)::boolean = FALSE
    OR s.started_at < sqlc.arg(to_ts)::timestamptz
  )
ORDER BY s.started_at DESC, s.id DESC
LIMIT sqlc.arg(lim)::int;

-- name: GetAgentLiveSession :one
SELECT
    s.id,
    s.project_id,
    p.name AS project_name,
    s.note,
    s.ticket_id,
    s.activity_type_id,
    COALESCE(a.name, '') AS activity_type_name,
    s.status,
    s.started_at,
    s.ended_at,
    s.target_duration_ms
FROM sessions s
JOIN projects p ON p.id = s.project_id AND p.user_id = s.user_id
LEFT JOIN activity_types a ON a.id = s.activity_type_id AND a.user_id = s.user_id
WHERE s.user_id = $1 AND s.status = 'active';

-- name: ListAgentSessionsInWindow :many
SELECT
    s.id,
    s.project_id,
    p.name AS project_name,
    s.note,
    s.ticket_id,
    s.activity_type_id,
    COALESCE(a.name, '') AS activity_type_name,
    s.status,
    s.started_at,
    s.ended_at,
    s.target_duration_ms
FROM sessions s
JOIN projects p ON p.id = s.project_id AND p.user_id = s.user_id
LEFT JOIN activity_types a ON a.id = s.activity_type_id AND a.user_id = s.user_id
WHERE s.user_id = sqlc.arg(user_id)
  AND s.started_at < sqlc.arg(to_ts)::timestamptz
  AND (s.ended_at IS NULL OR s.ended_at > sqlc.arg(from_ts)::timestamptz)
ORDER BY s.started_at ASC, s.id ASC
LIMIT 10001;

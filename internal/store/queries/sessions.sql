-- name: ListSessions :many
SELECT id, project_id, note, ticket_id, activity_type, tags, status,
       started_at, ended_at, paused_ms, paused_at, target_duration_ms
FROM sessions
WHERE user_id = $1
  AND (
    sqlc.arg(filter_statuses)::boolean = FALSE
    OR (sqlc.arg(want_active)::boolean AND status = 'active')
    OR (sqlc.arg(want_paused)::boolean AND status = 'paused')
    OR (sqlc.arg(want_stopped)::boolean AND status = 'stopped')
  )
ORDER BY started_at DESC
LIMIT NULLIF(sqlc.arg(lim)::int, 0);

-- name: GetSession :one
SELECT id, project_id, note, ticket_id, activity_type, tags, status,
       started_at, ended_at, paused_ms, paused_at, target_duration_ms
FROM sessions
WHERE user_id = $1 AND id = $2;

-- name: GetLiveSession :one
SELECT id, project_id, note, ticket_id, activity_type, tags, status,
       started_at, ended_at, paused_ms, paused_at, target_duration_ms
FROM sessions
WHERE user_id = $1 AND status IN ('active', 'paused');

-- name: InsertSession :one
INSERT INTO sessions (
    id, user_id, project_id, note, ticket_id, activity_type, tags, status,
    started_at, ended_at, paused_ms, paused_at, target_duration_ms
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8,
    $9, $10, $11, $12, $13
)
RETURNING id, project_id, note, ticket_id, activity_type, tags, status,
          started_at, ended_at, paused_ms, paused_at, target_duration_ms;

-- name: UpdateSession :one
UPDATE sessions
SET note = $3,
    ticket_id = $4,
    activity_type = $5,
    tags = $6,
    status = $7,
    started_at = $8,
    ended_at = $9,
    paused_ms = $10,
    paused_at = $11,
    target_duration_ms = $12
WHERE user_id = $1 AND id = $2
RETURNING id, project_id, note, ticket_id, activity_type, tags, status,
          started_at, ended_at, paused_ms, paused_at, target_duration_ms;

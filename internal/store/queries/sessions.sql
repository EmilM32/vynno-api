-- name: ListSessions :many
SELECT id, project_id, note, ticket_id, activity_type_id, tags, status,
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
SELECT id, project_id, note, ticket_id, activity_type_id, tags, status,
       started_at, ended_at, paused_ms, paused_at, target_duration_ms
FROM sessions
WHERE user_id = $1 AND id = $2;

-- name: GetLiveSession :one
SELECT id, project_id, note, ticket_id, activity_type_id, tags, status,
       started_at, ended_at, paused_ms, paused_at, target_duration_ms
FROM sessions
WHERE user_id = $1 AND status IN ('active', 'paused');

-- name: InsertSession :one
INSERT INTO sessions (
    id, user_id, project_id, note, ticket_id, activity_type_id, tags, status,
    started_at, ended_at, paused_ms, paused_at, target_duration_ms
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8,
    $9, $10, $11, $12, $13
)
RETURNING id, project_id, note, ticket_id, activity_type_id, tags, status,
          started_at, ended_at, paused_ms, paused_at, target_duration_ms;

-- name: UpdateSession :one
UPDATE sessions
SET project_id = $3,
    note = $4,
    ticket_id = $5,
    activity_type_id = $6,
    tags = $7,
    status = $8,
    started_at = $9,
    ended_at = $10,
    paused_ms = $11,
    paused_at = $12,
    target_duration_ms = $13
WHERE user_id = $1 AND id = $2
RETURNING id, project_id, note, ticket_id, activity_type_id, tags, status,
          started_at, ended_at, paused_ms, paused_at, target_duration_ms;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE user_id = $1 AND id = $2;

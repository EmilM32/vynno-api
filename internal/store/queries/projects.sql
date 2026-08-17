-- name: ListProjects :many
SELECT id, name, color, code, progress_percent, archived
FROM projects
WHERE user_id = $1
  AND (archived = FALSE OR sqlc.arg(include_archived)::boolean = TRUE)
ORDER BY name ASC;

-- name: GetProject :one
SELECT id, name, color, code, progress_percent, archived
FROM projects
WHERE user_id = $1 AND id = $2;

-- name: InsertProject :one
INSERT INTO projects (id, user_id, name, color, code, progress_percent, archived)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, name, color, code, progress_percent, archived;

-- name: UpdateProject :one
UPDATE projects
SET name = $3,
    color = $4,
    code = $5,
    progress_percent = $6,
    archived = $7,
    updated_at = now()
WHERE user_id = $1 AND id = $2
RETURNING id, name, color, code, progress_percent, archived;

-- name: DeleteProject :exec
DELETE FROM projects WHERE user_id = $1 AND id = $2;

-- name: CountActiveProjects :one
SELECT count(*)::bigint FROM projects WHERE user_id = $1 AND archived = FALSE;

-- name: CountProjectSessions :one
SELECT count(*)::bigint FROM sessions WHERE user_id = $1 AND project_id = $2;

-- name: CodeInUse :one
SELECT EXISTS(
    SELECT 1
    FROM projects
    WHERE user_id = $1
      AND code IS NOT NULL
      AND lower(code) = lower(sqlc.arg(code))
      AND id <> sqlc.arg(exclude_id)
)::boolean;

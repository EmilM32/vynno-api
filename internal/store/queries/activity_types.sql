-- name: ListActivityTypes :many
SELECT id, name, color
FROM activity_types
WHERE user_id = $1
ORDER BY lower(name) ASC;

-- name: GetActivityType :one
SELECT id, name, color
FROM activity_types
WHERE user_id = $1 AND id = $2;

-- name: InsertActivityType :one
INSERT INTO activity_types (id, user_id, name, color)
VALUES ($1, $2, $3, $4)
RETURNING id, name, color;

-- name: UpdateActivityType :one
UPDATE activity_types
SET name = $3,
    color = $4,
    updated_at = now()
WHERE user_id = $1 AND id = $2
RETURNING id, name, color;

-- name: DeleteActivityType :exec
DELETE FROM activity_types WHERE user_id = $1 AND id = $2;

-- name: CountActivityTypeSessions :one
SELECT count(*)::bigint FROM sessions WHERE user_id = $1 AND activity_type_id = $2;

-- name: ActivityTypeNameInUse :one
SELECT EXISTS(
    SELECT 1
    FROM activity_types
    WHERE user_id = $1
      AND lower(name) = lower(sqlc.arg(name))
      AND id <> sqlc.arg(exclude_id)
)::boolean;

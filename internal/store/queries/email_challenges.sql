-- name: GetEmailChallenge :one
SELECT email, purpose, code_hash, expires_at, attempt_count, sent_at, send_count, send_window_start
FROM email_challenges
WHERE email = $1 AND purpose = $2;

-- name: UpsertEmailChallenge :exec
INSERT INTO email_challenges (
    email, purpose, code_hash, expires_at, attempt_count, sent_at, send_count, send_window_start
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
ON CONFLICT (email, purpose) DO UPDATE SET
    code_hash = EXCLUDED.code_hash,
    expires_at = EXCLUDED.expires_at,
    attempt_count = EXCLUDED.attempt_count,
    sent_at = EXCLUDED.sent_at,
    send_count = EXCLUDED.send_count,
    send_window_start = EXCLUDED.send_window_start;

-- name: DeleteEmailChallenge :exec
DELETE FROM email_challenges WHERE email = $1 AND purpose = $2;

-- name: IncrementChallengeAttempts :one
UPDATE email_challenges
SET attempt_count = attempt_count + 1
WHERE email = $1 AND purpose = $2
RETURNING attempt_count;

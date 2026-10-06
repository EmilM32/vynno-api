-- name: GetEmailChallenge :one
SELECT email, purpose, code_hash, expires_at, attempt_count, sent_at, send_count, send_window_start, user_id
FROM email_challenges
WHERE email = $1 AND purpose = $2;

-- name: UpsertEmailChallenge :exec
INSERT INTO email_challenges (
    email, purpose, code_hash, expires_at, attempt_count, sent_at, send_count, send_window_start, user_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
ON CONFLICT (email, purpose) DO UPDATE SET
    code_hash = EXCLUDED.code_hash,
    expires_at = EXCLUDED.expires_at,
    attempt_count = EXCLUDED.attempt_count,
    sent_at = EXCLUDED.sent_at,
    send_count = EXCLUDED.send_count,
    send_window_start = EXCLUDED.send_window_start,
    user_id = EXCLUDED.user_id;

-- name: ReserveChallengeGuess :one
-- Takes one guess atomically before the code is compared. No row when the challenge
-- is missing, bound to another account, or already out of guesses.
UPDATE email_challenges
SET attempt_count = attempt_count + 1
WHERE email = $1
  AND purpose = $2
  AND user_id IS NOT DISTINCT FROM sqlc.narg(user_id)::uuid
  AND attempt_count < sqlc.arg(max_attempts)::int
RETURNING email, purpose, code_hash, expires_at, attempt_count, sent_at, send_count, send_window_start, user_id;

-- name: ConsumeEmailChallenge :execrows
-- Deletes the challenge only while it still holds the code that was compared.
DELETE FROM email_challenges
WHERE email = $1 AND purpose = $2 AND code_hash = $3;

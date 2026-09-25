-- TOTP credential and session queries

-- name: GetCredential :one
SELECT COALESCE(totp_secret, '') AS totp_secret, totp_enabled
FROM "user"
WHERE id = $1;

-- name: UpdateCredential :exec
UPDATE "user"
SET totp_secret = $2, totp_enabled = $3, updated_at = $4
WHERE id = $1;

-- name: MarkSessionVerified :exec
INSERT INTO totp_session (session_id, user_id, verified_at)
VALUES ($1, $2, $3)
ON CONFLICT (session_id) DO UPDATE SET verified_at = EXCLUDED.verified_at;

-- name: IsSessionVerified :one
SELECT EXISTS(
    SELECT 1 FROM totp_session
    WHERE session_id = $1 AND user_id = $2 AND verified_at > $3
);

-- name: DeleteUserSessionVerifications :exec
DELETE FROM totp_session WHERE user_id = $1;

-- name: DeleteSessionVerification :exec
DELETE FROM totp_session WHERE session_id = $1;

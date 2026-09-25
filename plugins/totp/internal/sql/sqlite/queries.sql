-- TOTP credential and session queries

-- name: GetCredential :one
SELECT COALESCE(totp_secret, '') AS totp_secret, totp_enabled
FROM "user"
WHERE id = ?;

-- name: UpdateCredential :exec
UPDATE "user"
SET totp_secret = ?, totp_enabled = ?, updated_at = ?
WHERE id = ?;

-- name: MarkSessionVerified :exec
INSERT INTO totp_session (session_id, user_id, verified_at)
VALUES (?, ?, ?)
ON CONFLICT (session_id) DO UPDATE SET verified_at = EXCLUDED.verified_at;

-- name: IsSessionVerified :one
SELECT EXISTS(
    SELECT 1 FROM totp_session
    WHERE session_id = ? AND user_id = ? AND verified_at > ?
);

-- name: DeleteUserSessionVerifications :exec
DELETE FROM totp_session WHERE user_id = ?;

-- name: DeleteSessionVerification :exec
DELETE FROM totp_session WHERE session_id = ?;

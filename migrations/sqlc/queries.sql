-- name: CreateUser :one
INSERT INTO auth.users (
    email,
    password_hash,
    created_at,
    updated_at
) VALUES (
    $1, $2, NOW(), NOW()
)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM auth.users
WHERE email = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM auth.users
WHERE id = $1 LIMIT 1;

-- name: UpdateUserPassword :exec
UPDATE auth.users
SET password_hash = $2, updated_at = NOW()
WHERE id = $1;

-- name: VerifyUserEmail :exec
UPDATE auth.users
SET email_verified = TRUE, updated_at = NOW()
WHERE id = $1;

-- name: CreateSession :one
INSERT INTO auth.sessions (
    user_id,
    token,
    refresh_token,
    ip_address,
    user_agent,
    expires_at,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, NOW()
)
RETURNING *;

-- name: GetSessionByToken :one
SELECT * FROM auth.sessions
WHERE token = $1 AND expires_at > NOW()
LIMIT 1;

-- name: GetSessionByRefreshToken :one
SELECT * FROM auth.sessions
WHERE refresh_token = $1
LIMIT 1;

-- name: DeleteSession :exec
DELETE FROM auth.sessions
WHERE token = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM auth.sessions
WHERE expires_at < NOW();

-- name: CreateOTP :one
INSERT INTO auth.otps (
    user_id,
    code,
    purpose,
    expires_at,
    created_at
) VALUES (
    $1, $2, $3, $4, NOW()
)
RETURNING *;

-- name: GetOTP :one
SELECT * FROM auth.otps
WHERE user_id = $1 AND purpose = $2 AND used = FALSE AND expires_at > NOW()
ORDER BY created_at DESC
LIMIT 1;

-- name: MarkOTPUsed :exec
UPDATE auth.otps
SET used = TRUE
WHERE id = $1;

-- name: InvalidateOTPs :exec
UPDATE auth.otps
SET used = TRUE
WHERE user_id = $1 AND purpose = $2 AND used = FALSE;

-- name: LinkOAuthProvider :one
INSERT INTO auth.oauth_providers (
    user_id,
    provider,
    provider_user_id,
    email,
    name,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, NOW(), NOW()
)
RETURNING *;

-- name: GetOAuthProvider :one
SELECT * FROM auth.oauth_providers
WHERE provider = $1 AND provider_user_id = $2
LIMIT 1;

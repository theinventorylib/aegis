-- User queries
-- name: CreateUser :exec
INSERT INTO "user" (id, avatar, name, email, created_at, updated_at, disabled) VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetUserByEmail :one
SELECT id, avatar, name, email, created_at, updated_at, disabled FROM "user" WHERE email = $1 AND disabled = 0;

-- name: GetUserByID :one
SELECT id, avatar, name, email, created_at, updated_at, disabled FROM "user" WHERE id = $1 AND disabled = 0;

-- name: UpdateUser :exec
UPDATE "user" SET avatar = $2, name = $3, email = $4, updated_at = $5, disabled = $6 WHERE id = $1;

-- name: DeleteUser :exec
UPDATE "user" SET disabled = 1, updated_at = $2 WHERE id = $1;

-- name: ListUsers :many
SELECT id, avatar, name, email, created_at, updated_at, disabled FROM "user" WHERE disabled = 0 ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM "user" WHERE disabled = 0;

-- Account queries
-- name: CreateAccount :exec
INSERT INTO accounts (id, user_id, provider, provider_account_id, password_hash, access_token, refresh_token, expires_at, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: GetAccountByID :one
SELECT id, user_id, provider, provider_account_id, password_hash, access_token, refresh_token, expires_at, created_at, updated_at FROM accounts WHERE id = $1;

-- name: GetAccountsByUserID :many
SELECT id, user_id, provider, provider_account_id, password_hash, access_token, refresh_token, expires_at, created_at, updated_at FROM accounts WHERE user_id = $1;

-- name: GetAccountByProvider :one
SELECT id, user_id, provider, provider_account_id, password_hash, access_token, refresh_token, expires_at, created_at, updated_at FROM accounts WHERE provider = $1 AND provider_account_id = $2;

-- name: UpdateAccount :exec
UPDATE accounts SET access_token = $2, refresh_token = $3, expires_at = $4, updated_at = $5 WHERE id = $1;

-- name: DeleteAccount :exec
DELETE FROM accounts WHERE id = $1;

-- Verification queries
-- name: CreateVerification :exec
INSERT INTO verification (id, identifier, token, type, expires_at, created_at) VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetVerificationByToken :one
SELECT id, identifier, token, type, expires_at, created_at FROM verification WHERE token = $1 AND expires_at > $2;

-- name: GetVerificationsByIdentifier :many
SELECT id, identifier, token, type, expires_at, created_at FROM verification WHERE identifier = $1 AND expires_at > $2;

-- name: InvalidateVerificationByIdentifier :exec
UPDATE verification SET expires_at = $3 WHERE identifier = $1 AND type = $2;

-- name: DeleteVerification :exec
DELETE FROM verification WHERE id = $1;

-- name: CleanupExpiredVerifications :exec
DELETE FROM verification WHERE expires_at <= $1;

-- Session queries
-- name: CreateSession :exec
INSERT INTO session (id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetSession :one
SELECT id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent FROM session WHERE id = $1 AND expires_at > $2;

-- name: GetSessionByToken :one
SELECT id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent FROM session WHERE token = $1 AND expires_at > $2;

-- name: GetSessionByRefreshToken :one
SELECT id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent FROM session WHERE refresh_token = $1 AND expires_at > $2;

-- name: GetSessionsByUserID :many
SELECT id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent FROM session WHERE user_id = $1 AND expires_at > $2 LIMIT $3 OFFSET $4;

-- name: CountSessionsByUserID :one
SELECT COUNT(*) FROM session WHERE user_id = $1 AND expires_at > $2;

-- name: UpdateSession :exec
UPDATE session SET refresh_token = $2, expires_at = $3 WHERE id = $1;

-- name: DeleteSession :exec
DELETE FROM session WHERE id = $1;

-- name: DeleteSessionsByUserID :exec
DELETE FROM session Where user_id = $1;

-- name: CleanupExpiredSessions :exec
DELETE FROM session WHERE expires_at <= $1;
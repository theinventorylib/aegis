-- User queries
-- name: CreateUser :exec
INSERT INTO "user" (id, avatar, name, email, created_at, updated_at, disabled) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetUserByEmail :one
SELECT id, avatar, name, email, created_at, updated_at, disabled FROM "user" WHERE email = ? AND disabled = 0;

-- name: GetUserByID :one
SELECT id, avatar, name, email, created_at, updated_at, disabled FROM "user" WHERE id = ? AND disabled = 0;

-- name: UpdateUser :exec
UPDATE "user" SET avatar = ?, name = ?, email = ?, updated_at = ?, disabled = ? WHERE id = ?;

-- name: DeleteUser :exec
UPDATE "user" SET disabled = 1, updated_at = ? WHERE id = ?;

-- name: ListUsers :many
SELECT id, avatar, name, email, created_at, updated_at, disabled FROM "user" WHERE disabled = 0 ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountUsers :one
SELECT COUNT(*) FROM "user" WHERE disabled = 0;

-- Account queries
-- name: CreateAccount :exec
INSERT INTO accounts (id, user_id, provider, provider_account_id, password_hash, access_token, refresh_token, expires_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAccountByID :one
SELECT id, user_id, provider, provider_account_id, password_hash, access_token, refresh_token, expires_at, created_at, updated_at FROM accounts WHERE id = ?;

-- name: GetAccountsByUserID :many
SELECT id, user_id, provider, provider_account_id, password_hash, access_token, refresh_token, expires_at, created_at, updated_at FROM accounts WHERE user_id = ?;

-- name: GetAccountByProvider :one
SELECT id, user_id, provider, provider_account_id, password_hash, access_token, refresh_token, expires_at, created_at, updated_at FROM accounts WHERE provider = ? AND provider_account_id = ?;

-- name: UpdateAccount :exec
UPDATE accounts SET access_token = ?, refresh_token = ?, expires_at = ?, updated_at = ? WHERE id = ?;

-- name: DeleteAccount :exec
DELETE FROM accounts WHERE id = ?;

-- Verification queries
-- name: CreateVerification :exec
INSERT INTO verification (id, identifier, token, type, expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?);

-- name: GetVerificationByToken :one
SELECT id, identifier, token, type, expires_at, created_at FROM verification WHERE token = ? AND expires_at > ?;

-- name: GetVerificationsByIdentifier :many
SELECT id, identifier, token, type, expires_at, created_at FROM verification WHERE identifier = ? AND expires_at > ?;

-- name: InvalidateVerificationByIdentifier :exec
UPDATE verification SET expires_at = ? WHERE identifier = ? AND type = ?;

-- name: DeleteVerification :exec
DELETE FROM verification WHERE id = ?;

-- name: CleanupExpiredVerifications :exec
DELETE FROM verification WHERE expires_at <= ?;

-- Session queries
-- name: CreateSession :exec
INSERT INTO session (id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetSession :one
SELECT id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent FROM session WHERE id = ? AND expires_at > ?;

-- name: GetSessionByToken :one
SELECT id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent FROM session WHERE token = ? AND expires_at > ?;

-- name: GetSessionByRefreshToken :one
SELECT id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent FROM session WHERE refresh_token = ? AND expires_at > ?;

-- name: GetSessionsByUserID :many
SELECT id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent FROM session WHERE user_id = ? AND expires_at > ? LIMIT ? OFFSET ?;

-- name: CountSessionsByUserID :one
SELECT COUNT(*) FROM session WHERE user_id = ? AND expires_at > ?;

-- name: UpdateSession :exec
UPDATE session SET refresh_token = ?, expires_at = ? WHERE id = ?;

-- name: DeleteSession :exec
DELETE FROM session WHERE id = ?;

-- name: DeleteSessionsByUserID :exec
DELETE FROM session Where user_id = ?;

-- name: CleanupExpiredSessions :exec
DELETE FROM session WHERE expires_at <= ?;

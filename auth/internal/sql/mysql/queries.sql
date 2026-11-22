-- User queries
-- name: CreateUser :exec
INSERT INTO `user` (id, avatar, name, email, created_at, updated_at, disabled) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetUserByEmail :one
SELECT id, avatar, name, email, created_at, updated_at, disabled FROM `user` WHERE email = ?;

-- name: GetUserByID :one
SELECT id, avatar, name, email, created_at, updated_at, disabled FROM `user` WHERE id = ?;

-- name: UpdateUser :exec
UPDATE `user` SET avatar = ?, name = ?, email = ?, updated_at = ?, disabled = ? WHERE id = ?;

-- name: DeleteUser :exec
UPDATE `user` SET disabled = 1, updated_at = ? WHERE id = ?;

-- name: ListUsers :many
SELECT id, avatar, name, email, created_at, updated_at, disabled FROM `user` WHERE disabled = 0 ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountUsers :one
SELECT COUNT(*) FROM `user` WHERE disabled = 0;

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
UPDATE accounts SET password_hash = COALESCE(sqlc.narg('password_hash'), password_hash), access_token = sqlc.narg('access_token'), refresh_token = sqlc.narg('refresh_token'), expires_at = sqlc.narg('expires_at'), updated_at = sqlc.arg('updated_at') WHERE id = sqlc.arg('id');

-- name: UpdateAccountPassword :exec
UPDATE accounts SET password_hash = sqlc.narg('password_hash'), updated_at = sqlc.arg('updated_at') WHERE id = sqlc.arg('id');

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
DELETE FROM session WHERE user_id = ?;

-- name: CleanupExpiredSessions :exec
DELETE FROM session WHERE expires_at <= ?;

-- Impersonation queries
-- name: CreateImpersonation :exec
INSERT INTO session_impersonation (session_id, actor_user_id, organization_id, created_at, expires_at) VALUES (?, ?, ?, ?, ?);

-- name: GetImpersonationBySession :one
SELECT session_id, actor_user_id, organization_id, created_at, expires_at FROM session_impersonation WHERE session_id = ?;

-- name: DeleteImpersonation :exec
DELETE FROM session_impersonation WHERE session_id = ?;

-- name: ListImpersonationsByActor :many
SELECT session_id, actor_user_id, organization_id, created_at, expires_at FROM session_impersonation WHERE actor_user_id = ?;

-- name: CleanupExpiredImpersonations :exec
DELETE FROM session_impersonation WHERE expires_at <= ?;

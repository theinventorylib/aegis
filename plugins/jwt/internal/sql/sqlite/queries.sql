-- JWT Key queries

-- name: GetCurrentJWK :one
SELECT key_data FROM jwks
WHERE algorithm = ? AND use = ? AND (expires_at IS NULL OR expires_at > datetime('now'))
ORDER BY created_at DESC LIMIT 1;

-- name: StoreJWK :exec
INSERT INTO jwks (kid, key_data, algorithm, use, created_at, expires_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: DeleteExpiredJWKS :exec
DELETE FROM jwks WHERE expires_at IS NOT NULL AND expires_at < datetime('now');

-- name: GetAllCurrentJWKS :many
SELECT kid, key_data, algorithm, use, created_at, expires_at FROM jwks
WHERE expires_at IS NULL OR expires_at > datetime('now')
ORDER BY created_at DESC;

-- name: CleanupOldKeys :exec
DELETE FROM jwks WHERE created_at < ?;

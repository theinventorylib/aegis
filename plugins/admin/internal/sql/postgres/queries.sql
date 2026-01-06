-- User queries
-- name: CreateUser :exec
INSERT INTO "user" (id, avatar, name, email, created_at, updated_at, disabled, role, banned, ban_reason, ban_expiry, ban_counter) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: GetUserByEmail :one
SELECT id, avatar, name, email, created_at, updated_at, disabled, role, banned, ban_reason, ban_expiry, ban_counter FROM "user" WHERE email = $1;

-- name: GetUserByID :one
SELECT id, avatar, name, email, created_at, updated_at, disabled, role, banned, ban_reason, ban_expiry, ban_counter FROM "user" WHERE id = $1;

-- name: UpdateUser :exec
UPDATE "user" SET avatar = $2, name = $3, email = $4, updated_at = $5, disabled = $6 WHERE id = $1;

-- name: DeleteUser :exec
UPDATE "user" SET disabled = 1, updated_at = $2 WHERE id = $1;

-- name: ListUsers :many
SELECT id, avatar, name, email, created_at, updated_at, disabled, role, banned, ban_reason, ban_expiry, ban_counter FROM "user" ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: ListUsersRaw :many
SELECT id, created_at, updated_at, COALESCE(email, '') as email, COALESCE(role, 'user') as role, disabled FROM "user" ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: GetUserRaw :one
SELECT id, created_at, updated_at, COALESCE(email, '') as email, COALESCE(role, 'user') as role, disabled FROM "user" WHERE id = $1;

-- name: CountUsers :one
SELECT COUNT(*) FROM "user";

-- name: GetRole :one
SELECT COALESCE(role, 'user') FROM "user" WHERE id = $1;

-- name: GetUsersByRole :many
SELECT id, avatar, name, email, created_at, updated_at, disabled, role, banned, ban_reason, ban_expiry, ban_counter FROM "user" WHERE role = $1 ORDER BY created_at DESC;

-- name: GetAdmins :many
SELECT id, avatar, name, email, created_at, updated_at, disabled, role, banned, ban_reason, ban_expiry, ban_counter FROM "user" WHERE role = 'admin' ORDER BY created_at DESC;

-- name: UpdateUserRole :exec
UPDATE "user" SET role = $2, updated_at = $3 WHERE id = $1;

-- name: BanUser :exec
UPDATE "user" SET banned = 1, ban_reason = $2, ban_expiry = $3, ban_counter = ban_counter + 1, updated_at = $4 WHERE id = $1;

-- name: UnbanUser :exec
UPDATE "user" SET banned = 0, ban_reason = NULL, ban_expiry = NULL, updated_at = $2 WHERE id = $1;
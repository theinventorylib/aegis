-- User queries
-- name: CreateUser :exec
INSERT INTO "user" (id, avatar, name, email, created_at, updated_at, disabled, role, banned, ban_reason, ban_expiry, ban_counter) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetUserByEmail :one
SELECT id, avatar, name, email, created_at, updated_at, disabled, role, banned, ban_reason, ban_expiry, ban_counter FROM "user" WHERE email = ?;

-- name: GetUserByID :one
SELECT id, avatar, name, email, created_at, updated_at, disabled, role, banned, ban_reason, ban_expiry, ban_counter FROM "user" WHERE id = ?;

-- name: UpdateUser :exec
UPDATE "user" SET avatar = ?, name = ?, email = ?, updated_at = ?, disabled = ? WHERE id = ?;

-- name: DeleteUser :exec
UPDATE "user" SET disabled = 1, updated_at = ? WHERE id = ?;

-- name: ListUsers :many
SELECT id, avatar, name, email, created_at, updated_at, disabled, role, banned, ban_reason, ban_expiry, ban_counter FROM "user" ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: ListUsersRaw :many
SELECT id, created_at, updated_at, COALESCE(email, '') as email, COALESCE(role, 'user') as role, disabled FROM "user" ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: GetUserRaw :one
SELECT id, created_at, updated_at, COALESCE(email, '') as email, COALESCE(role, 'user') as role, disabled FROM "user" WHERE id = ?;

-- name: CountUsers :one
SELECT COUNT(*) FROM "user";

-- name: GetRole :one
SELECT COALESCE(role, 'user') FROM "user" WHERE id = ?;

-- name: GetUsersByRole :many
SELECT id, avatar, name, email, created_at, updated_at, disabled, role, banned, ban_reason, ban_expiry, ban_counter FROM "user" WHERE role = ? ORDER BY created_at DESC;

-- name: GetAdmins :many
SELECT id, avatar, name, email, created_at, updated_at, disabled, role, banned, ban_reason, ban_expiry, ban_counter FROM "user" WHERE role = 'admin' ORDER BY created_at DESC;

-- name: UpdateUserRole :exec
UPDATE "user" SET role = ?, updated_at = ? WHERE id = ?;

-- name: BanUser :exec
UPDATE "user" SET banned = 1, ban_reason = ?, ban_expiry = ?, ban_counter = ban_counter + 1, updated_at = ? WHERE id = ?;

-- name: UnbanUser :exec
UPDATE "user" SET banned = 0, ban_reason = NULL, ban_expiry = NULL, updated_at = ? WHERE id = ?;

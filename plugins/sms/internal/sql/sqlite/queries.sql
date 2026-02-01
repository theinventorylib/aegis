-- SMS User Phone queries

-- name: CreateUser :exec
INSERT INTO "user" (id, avatar, name, email, created_at, updated_at, disabled, phone_number, phone_verified)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetUserByID :one
SELECT u.id, u.avatar, u.name, u.email, u.created_at, u.updated_at, u.disabled,
       u.phone_number, u.phone_verified
FROM "user" u
WHERE u.id = ?;

-- name: GetUserByPhone :one
SELECT u.id, u.avatar, u.name, u.email, u.created_at, u.updated_at, u.disabled,
       u.phone_number, u.phone_verified
FROM "user" u
WHERE u.phone_number = ?;

-- name: UpdateUserPhone :exec
UPDATE "user"
SET phone_number = ?, phone_verified = ?, updated_at = ?
WHERE id = ?;

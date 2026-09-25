-- Organizations queries

-- name: CreateOrganization :exec
INSERT INTO organization (id, name, slug, created_at, updated_at) VALUES (?, ?, ?, ?, ?);

-- name: GetOrganization :one
SELECT id, name, slug, created_at, updated_at FROM organization WHERE id = ? AND disabled = 0;

-- name: GetOrganizationBySlug :one
SELECT id, name, slug, created_at, updated_at FROM organization WHERE slug = ? AND disabled = 0;

-- name: UpdateOrganization :exec
UPDATE organization SET name = ?, slug = ?, updated_at = ? WHERE id = ?;

-- name: DeleteOrganization :exec
UPDATE organization SET disabled = 1, updated_at = ? WHERE id = ?;

-- name: ListUserOrganizations :many
SELECT o.id, o.name, o.slug, o.created_at, o.updated_at
FROM organization o
JOIN members uo ON o.id = uo.organization_id
WHERE uo.user_id = ? AND o.disabled = 0
ORDER BY o.created_at DESC LIMIT ? OFFSET ?;

-- name: CountUserOrganizations :one
SELECT COUNT(*)
FROM organization o
JOIN members uo ON o.id = uo.organization_id
WHERE uo.user_id = ? AND o.disabled = 0;

-- User Organizations queries

-- name: CreateMember :exec
INSERT INTO members (id, user_id, organization_id, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?);

-- name: GetMember :one
SELECT id, user_id, organization_id, role, created_at, updated_at FROM members WHERE user_id = ? AND organization_id = ?;

-- name: IsOrganizationMember :one
SELECT COUNT(*) > 0 as is_member FROM members WHERE user_id = ? AND organization_id = ?;

-- name: IsOwnerOrAdmin :one
SELECT COUNT(*) > 0 as is_owner_admin FROM members WHERE user_id = ? AND organization_id = ? AND role IN ('owner', 'admin');

-- name: IsOwner :one
SELECT COUNT(*) > 0 as is_owner FROM members WHERE user_id = ? AND organization_id = ? AND role = 'owner';

-- name: UpdateMemberRole :exec
UPDATE members SET role = ?, updated_at = ? WHERE user_id = ? AND organization_id = ?;

-- name: RemoveMember :exec
DELETE FROM members WHERE user_id = ? AND organization_id = ?;

-- name: ListOrganizationMembers :many
SELECT id, user_id, organization_id, role, created_at, updated_at FROM members WHERE organization_id = ? ORDER BY created_at ASC LIMIT ? OFFSET ?;

-- name: CountOrganizationMembers :one
SELECT COUNT(*) FROM members WHERE organization_id = ?;

-- Teams queries

-- name: CreateTeam :exec
INSERT INTO team (id, organization_id, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?);

-- name: GetTeam :one
SELECT id, organization_id, name, description, created_at, updated_at FROM team WHERE id = ?;

-- name: ListTeams :many
SELECT id, organization_id, name, description, created_at, updated_at FROM team WHERE organization_id = ? ORDER BY created_at ASC LIMIT ? OFFSET ?;

-- name: CountTeams :one
SELECT COUNT(*) FROM team WHERE organization_id = ?;

-- name: UpdateTeam :exec
UPDATE team SET name = ?, description = ?, updated_at = ? WHERE id = ?;

-- name: DeleteTeam :exec
DELETE FROM team WHERE id = ?;

-- Team Members queries

-- name: CreateTeamMember :exec
INSERT INTO team_member (id, team_id, user_id, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?);

-- name: GetTeamMember :one
SELECT id, team_id, user_id, role, created_at, updated_at FROM team_member WHERE team_id = ? AND user_id = ?;

-- name: ListTeamMembers :many
SELECT id, team_id, user_id, role, created_at, updated_at FROM team_member WHERE team_id = ? ORDER BY created_at ASC LIMIT ? OFFSET ?;

-- name: CountTeamMembers :one
SELECT COUNT(*) FROM team_member WHERE team_id = ?;

-- name: UpdateTeamMemberRole :exec
UPDATE team_member SET role = ?, updated_at = ? WHERE team_id = ? AND user_id = ?;

-- name: RemoveTeamMember :exec
DELETE FROM team_member WHERE team_id = ? AND user_id = ?;

-- Invitation queries

-- name: CreateInvitation :exec
INSERT INTO invitation (id, organization_id, team_id, email, role, inviter_id, token_hash, status, expires_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetInvitationByID :one
SELECT id, organization_id, team_id, email, role, inviter_id, token_hash, status, expires_at, created_at, updated_at
FROM invitation WHERE id = ?;

-- name: GetInvitationByTokenHash :one
SELECT id, organization_id, team_id, email, role, inviter_id, token_hash, status, expires_at, created_at, updated_at
FROM invitation WHERE token_hash = ?;

-- name: ListInvitations :many
SELECT id, organization_id, team_id, email, role, inviter_id, token_hash, status, expires_at, created_at, updated_at
FROM invitation
WHERE organization_id = ?
  AND (? = '' OR team_id = ?)
ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountInvitations :one
SELECT COUNT(*)
FROM invitation
WHERE organization_id = ?
  AND (? = '' OR team_id = ?);

-- name: UpdateInvitationStatus :exec
UPDATE invitation SET status = ?, updated_at = ? WHERE id = ?;

-- name: DeleteInvitation :exec
DELETE FROM invitation WHERE id = ?;

-- name: ListMemberPermissionOverrides :many
SELECT id, organization_id, user_id, permission, effect, created_at, updated_at
FROM member_permission_override
WHERE organization_id = ? AND user_id = ?
ORDER BY permission;

-- name: CreateMemberPermissionOverride :exec
INSERT INTO member_permission_override (
    id, organization_id, user_id, permission, effect, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: DeleteMemberPermissionOverrides :exec
DELETE FROM member_permission_override WHERE organization_id = ? AND user_id = ?;

-- name: ListOrganizationRoles :many
SELECT id, organization_id, name, permissions, created_at, updated_at
FROM organization_role
WHERE organization_id = ?
ORDER BY name;

-- name: GetOrganizationRole :one
SELECT id, organization_id, name, permissions, created_at, updated_at
FROM organization_role
WHERE organization_id = ? AND name = ?;

-- name: CreateOrganizationRole :exec
INSERT INTO organization_role (id, organization_id, name, permissions, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateOrganizationRole :exec
UPDATE organization_role
SET permissions = ?, updated_at = ?
WHERE organization_id = ? AND name = ?;

-- name: DeleteOrganizationRole :exec
DELETE FROM organization_role WHERE organization_id = ? AND name = ?;

-- name: CountOrganizationMembersWithRole :one
SELECT COUNT(*) FROM members WHERE organization_id = ? AND role = ?;

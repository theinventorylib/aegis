-- Organizations queries

-- name: CreateOrganization :exec
INSERT INTO organization (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5);

-- name: GetOrganization :one
SELECT id, name, slug, created_at, updated_at FROM organization WHERE id = $1 AND disabled = 0;

-- name: GetOrganizationBySlug :one
SELECT id, name, slug, created_at, updated_at FROM organization WHERE slug = $1 AND disabled = 0;

-- name: UpdateOrganization :exec
UPDATE organization SET name = $2, slug = $3, updated_at = $4 WHERE id = $1;

-- name: DeleteOrganization :exec
UPDATE organization SET disabled = 1, updated_at = $2 WHERE id = $1;

-- name: ListUserOrganizations :many
SELECT o.id, o.name, o.slug, o.created_at, o.updated_at
FROM organization o
JOIN members uo ON o.id = uo.organization_id
WHERE uo.user_id = $1 AND o.disabled = 0
ORDER BY o.created_at DESC;

-- User Organizations queries

-- name: CreateMember :exec
INSERT INTO members (id, user_id, organization_id, role, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetMember :one
SELECT id, user_id, organization_id, role, created_at, updated_at FROM members WHERE user_id = $1 AND organization_id = $2;

-- name: IsOrganizationMember :one
SELECT COUNT(*) > 0 as is_member FROM members WHERE user_id = $1 AND organization_id = $2;

-- name: IsOwnerOrAdmin :one
SELECT COUNT(*) > 0 as is_owner_admin FROM members WHERE user_id = $1 AND organization_id = $2 AND role IN ('owner', 'admin');

-- name: IsOwner :one
SELECT COUNT(*) > 0 as is_owner FROM members WHERE user_id = $1 AND organization_id = $2 AND role = 'owner';

-- name: UpdateMemberRole :exec
UPDATE members SET role = $3, updated_at = $4 WHERE user_id = $1 AND organization_id = $2;

-- name: RemoveMember :exec
DELETE FROM members WHERE user_id = $1 AND organization_id = $2;

-- name: ListOrganizationMembers :many
SELECT id, user_id, organization_id, role, created_at, updated_at FROM members WHERE organization_id = $1 ORDER BY created_at ASC;

-- Teams queries

-- name: CreateTeam :exec
INSERT INTO team (id, organization_id, name, description, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetTeam :one
SELECT id, organization_id, name, description, created_at, updated_at FROM team WHERE id = $1;

-- name: ListTeams :many
SELECT id, organization_id, name, description, created_at, updated_at FROM team WHERE organization_id = $1 ORDER BY created_at ASC;

-- name: UpdateTeam :exec
UPDATE team SET name = $2, description = $3, updated_at = $4 WHERE id = $1;

-- name: DeleteTeam :exec
DELETE FROM team WHERE id = $1;

-- Team Members queries

-- name: CreateTeamMember :exec
INSERT INTO team_member (id, team_id, user_id, role, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetTeamMember :one
SELECT id, team_id, user_id, role, created_at, updated_at FROM team_member WHERE team_id = $1 AND user_id = $2;

-- name: ListTeamMembers :many
SELECT id, team_id, user_id, role, created_at, updated_at FROM team_member WHERE team_id = $1 ORDER BY created_at ASC;

-- name: UpdateTeamMemberRole :exec
UPDATE team_member SET role = $3, updated_at = $4 WHERE team_id = $1 AND user_id = $2;

-- name: RemoveTeamMember :exec
DELETE FROM team_member WHERE team_id = $1 AND user_id = $2;
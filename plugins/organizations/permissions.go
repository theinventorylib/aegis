package organizations

import (
	"context"
	"database/sql"
	"errors"

	orgtypes "github.com/theinventorylib/aegis/v2/plugins/organizations/types"
)

// Permission is a capability that a role can grant within an organization or
// team. Applications check permissions rather than role names, so custom roles
// are first-class: define a role, grant it permissions, and the built-in
// handlers honor it.
type Permission string

const (
	// PermOrgView allows viewing the organization and its members/teams.
	PermOrgView Permission = "org:view"
	// PermOrgManage allows updating organization settings.
	PermOrgManage Permission = "org:manage"
	// PermOrgDelete allows deleting the organization.
	PermOrgDelete Permission = "org:delete"

	// PermMemberView allows listing organization members.
	PermMemberView Permission = "member:view"
	// PermMemberManage allows adding and removing organization members.
	PermMemberManage Permission = "member:manage"
	// PermMemberAssignRoles allows changing a member's organization role.
	PermMemberAssignRoles Permission = "member:assign_roles"

	// PermTeamView allows viewing teams and their members.
	PermTeamView Permission = "team:view"
	// PermTeamManage allows creating, updating, and deleting teams.
	PermTeamManage Permission = "team:manage"
	// PermTeamMemberManage allows adding, updating, and removing team members.
	PermTeamMemberManage Permission = "team_member:manage"

	// PermInvitationManage allows creating, listing, and canceling invitations.
	PermInvitationManage Permission = "invitation:manage"
)

// RoleDefinition describes an assignable organization or team role and the
// permissions it grants.
type RoleDefinition struct {
	// Permissions granted to members holding this role.
	Permissions []Permission
}

// Allows reports whether the role definition grants perm.
func (d RoleDefinition) Allows(perm Permission) bool {
	for _, p := range d.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}

// defaultOrgRoles are the built-in organization roles. Config.OrgRoles merges
// over these (same key replaces the definition).
func defaultOrgRoles() map[string]RoleDefinition {
	admin := []Permission{
		PermOrgView, PermOrgManage,
		PermMemberView, PermMemberManage, PermMemberAssignRoles,
		PermTeamView, PermTeamManage, PermTeamMemberManage,
		PermInvitationManage,
	}
	owner := append([]Permission{PermOrgDelete}, admin...)
	return map[string]RoleDefinition{
		orgtypes.RoleOwner:  {Permissions: owner},
		orgtypes.RoleAdmin:  {Permissions: admin},
		orgtypes.RoleMember: {Permissions: []Permission{PermOrgView, PermMemberView, PermTeamView}},
	}
}

// defaultTeamRoles are the built-in team roles. Config.TeamRoles merges over
// these (same key replaces the definition).
func defaultTeamRoles() map[string]RoleDefinition {
	return map[string]RoleDefinition{
		orgtypes.RoleTeamLead: {Permissions: []Permission{PermTeamView, PermTeamMemberManage}},
		orgtypes.RoleMember:   {Permissions: []Permission{PermTeamView}},
	}
}

// resolveRoles merges overrides on top of the defaults. Replacing a built-in
// key overrides that role's permissions; new keys add custom roles.
func resolveRoles(defaults, overrides map[string]RoleDefinition) map[string]RoleDefinition {
	out := make(map[string]RoleDefinition, len(defaults)+len(overrides))
	for k, v := range defaults {
		out[k] = v
	}
	for k, v := range overrides {
		out[k] = v
	}
	return out
}

// OrgRoles returns the resolved organization role definitions (built-ins merged
// with Config.OrgRoles). Mutating the returned map does not affect the plugin.
func (p *Plugin) OrgRoles() map[string]RoleDefinition {
	return cloneRoles(p.orgRoles)
}

// TeamRoles returns the resolved team role definitions (built-ins merged with
// Config.TeamRoles). Mutating the returned map does not affect the plugin.
func (p *Plugin) TeamRoles() map[string]RoleDefinition {
	return cloneRoles(p.teamRoles)
}

func cloneRoles(in map[string]RoleDefinition) map[string]RoleDefinition {
	out := make(map[string]RoleDefinition, len(in))
	for k, v := range in {
		out[k] = RoleDefinition{Permissions: append([]Permission(nil), v.Permissions...)}
	}
	return out
}

// HasOrgPermission reports whether the user's role in the organization grants
// perm. Users who are not members, or whose role is unknown, have no
// permissions. A store lookup error other than "member not found" is returned.
func (p *Plugin) HasOrgPermission(ctx context.Context, userID, orgID string, perm Permission) (bool, error) {
	m, err := p.store.GetMember(ctx, userID, orgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return p.orgRoles[m.Role].Allows(perm), nil
}

// HasTeamPermission reports whether the user's role in the team grants perm.
// Users who are not team members, or whose role is unknown, have no
// permissions. A store lookup error other than "member not found" is returned.
func (p *Plugin) HasTeamPermission(ctx context.Context, userID, teamID string, perm Permission) (bool, error) {
	m, err := p.store.GetTeamMember(ctx, teamID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return p.teamRoles[m.Role].Allows(perm), nil
}

// hasOrgPermissionForUser is the boolean, error-collapsed form used by
// handlers: any lookup failure is treated as "no permission" (fail closed).
func (p *Plugin) hasOrgPermissionForUser(ctx context.Context, userID, orgID string, perm Permission) bool {
	ok, err := p.HasOrgPermission(ctx, userID, orgID, perm)
	return err == nil && ok
}

// canManageTeamMembers reports whether the user may modify a team's membership:
// either their org role grants PermTeamMemberManage (e.g. owner/admin), or
// their team role does (e.g. team lead).
func (p *Plugin) canManageTeamMembers(ctx context.Context, userID, teamID, orgID string) bool {
	if p.hasOrgPermissionForUser(ctx, userID, orgID, PermTeamMemberManage) {
		return true
	}
	ok, err := p.HasTeamPermission(ctx, userID, teamID, PermTeamMemberManage)
	return err == nil && ok
}

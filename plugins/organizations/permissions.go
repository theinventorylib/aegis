package organizations

import (
	"context"
	"database/sql"
	"errors"
	"slices"

	"github.com/theinventorylib/aegis/v2/core"
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

// frameworkPermissions is the set of permissions Aegis' own handlers enforce.
// Grants of these are capped by what the actor holds; app-defined permission
// strings are not capped because the application owns their meaning.
var frameworkPermissions = map[Permission]struct{}{
	PermOrgView: {}, PermOrgManage: {}, PermOrgDelete: {},
	PermMemberView: {}, PermMemberManage: {}, PermMemberAssignRoles: {},
	PermTeamView: {}, PermTeamManage: {}, PermTeamMemberManage: {},
	PermInvitationManage: {},
}

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

// HasOrgPermission reports whether the user may exercise perm in the
// organization. The role's permissions are the baseline; per-member overrides
// adjust them: a deny always wins, a grant adds a permission the role does not
// carry. Users who are not members, or whose role is unknown, have no
// permissions. A store lookup error other than "member not found" is returned.
func (p *Plugin) HasOrgPermission(ctx context.Context, userID, orgID string, perm Permission) (bool, error) {
	m, err := p.store.GetMember(ctx, userID, orgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	allowed, err := p.roleAllows(ctx, orgID, m.Role, perm)
	if err != nil {
		return false, err
	}

	overrides, err := p.store.ListMemberPermissionOverrides(ctx, orgID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return allowed, nil
		}
		return false, err
	}
	for _, o := range overrides {
		if Permission(o.Permission) != perm {
			continue
		}
		switch o.Effect {
		case orgtypes.PermissionEffectDeny:
			return false, nil
		case orgtypes.PermissionEffectGrant:
			return true, nil
		}
	}
	return allowed, nil
}

// ungrantablePermission returns the first permission in perms that actorID may
// not grant, or "" when every permission is grantable. A framework permission
// the actor does not hold is refused, so a member who can assign permissions
// cannot mint authority they were never given (for example an admin granting
// org:delete, which the built-in admin role deliberately lacks). App-defined
// permission strings are always grantable.
func (p *Plugin) ungrantablePermission(ctx context.Context, actorID, orgID string, perms []Permission) (Permission, error) {
	var held map[Permission]bool
	for _, raw := range perms {
		// Normalize exactly like the write path does: a padded or control-char
		// spelling (" org:delete", "org:\x00delete") must be capped by the
		// same rule as the canonical string, or it would slip through here and
		// be stored sanitized.
		perm := Permission(core.SanitizeString(string(raw), nil))
		if _, known := frameworkPermissions[perm]; !known {
			continue
		}
		if held == nil {
			// Resolve the actor's effective set once rather than per permission.
			effective, err := p.GetMemberPermissions(ctx, actorID, orgID)
			if err != nil {
				return "", err
			}
			held = make(map[Permission]bool, len(effective.Permissions))
			for _, h := range effective.Permissions {
				held[h] = true
			}
		}
		if !held[perm] {
			return perm, nil
		}
	}
	return "", nil
}

// MemberPermissions describes a member's effective authorization: the role,
// the raw overrides, and the resolved permission set.
type MemberPermissions struct {
	Role        string                              `json:"role"`
	Overrides   []orgtypes.MemberPermissionOverride `json:"overrides"`
	Permissions []Permission                        `json:"permissions"`
}

// MemberPermissionsEntry is one row of the members-permissions listing: a
// member's user ID plus the effective authorization resolved for them. The
// embedded MemberPermissions fields are flattened into the JSON object.
type MemberPermissionsEntry struct {
	UserID string `json:"userId"`
	MemberPermissions
}

// GetMemberPermissions resolves a member's effective permissions: the role's
// permissions with per-member overrides applied (deny wins over grant, grant
// wins over the role). Returns sql.ErrNoRows when the user is not a member.
func (p *Plugin) GetMemberPermissions(ctx context.Context, userID, orgID string) (MemberPermissions, error) {
	m, err := p.store.GetMember(ctx, userID, orgID)
	if err != nil {
		return MemberPermissions{}, err
	}
	overrides, err := p.store.ListMemberPermissionOverrides(ctx, orgID, userID)
	if err != nil {
		return MemberPermissions{}, err
	}

	rolePerms, err := p.rolePermissions(ctx, orgID, m.Role)
	if err != nil {
		return MemberPermissions{}, err
	}
	effective := make(map[Permission]bool, len(rolePerms)+len(overrides))
	for _, perm := range rolePerms {
		effective[perm] = true
	}
	for _, o := range overrides {
		switch o.Effect {
		case orgtypes.PermissionEffectDeny:
			delete(effective, Permission(o.Permission))
		case orgtypes.PermissionEffectGrant:
			effective[Permission(o.Permission)] = true
		}
	}

	perms := make([]Permission, 0, len(effective))
	for perm := range effective {
		perms = append(perms, perm)
	}
	slices.Sort(perms)

	return MemberPermissions{Role: m.Role, Overrides: overrides, Permissions: perms}, nil
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

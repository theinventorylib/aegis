package organizations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

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
// strings are capped only when the catalog marks them Sensitive (see
// PermissionDefinition).
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

// ErrUnknownPermission means a permission string is not part of the
// configured permission catalog (Config.Permissions). It is only returned
// when a catalog is configured.
var ErrUnknownPermission = errors.New("unknown permission")

// PermissionDefinition describes an app-defined permission in the optional
// permission catalog (Config.Permissions).
type PermissionDefinition struct {
	// Description is human-readable text for UIs.
	Description string
	// Module groups related permissions in UIs (for example "billing").
	Module string
	// Sensitive marks an app-defined permission as capped: it may only be
	// granted (through roles, overrides, or assignments) by members who hold
	// it themselves. Framework permissions are always capped regardless of
	// this flag, so registering one with Sensitive false cannot loosen it.
	Sensitive bool
}

// PermissionInfo is one entry of the resolved permission catalog as returned
// by PermissionCatalog.
type PermissionInfo struct {
	Permission  Permission
	Description string
	Module      string
	Sensitive   bool
	Framework   bool
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
// organization. The membership role's permissions, plus those of any
// additional organization-wide assignment, are the baseline; per-member
// overrides adjust them: a deny always wins, a grant adds a permission the
// role does not carry. Users who are not members, or whose role is unknown,
// have no permissions. A store lookup error other than "member not found" is
// returned.
//
// Scoped role assignments do NOT count here: this is the organization-wide
// check. Use HasOrgPermissionOn for scoped resources.
func (p *Plugin) HasOrgPermission(ctx context.Context, userID, orgID string, perm Permission) (bool, error) {
	m, err := p.store.GetMember(ctx, userID, orgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	allowed, err := p.orgPermissionDecision(ctx, orgID, userID, m.Role, perm, "", "")
	if err != nil {
		return false, err
	}
	return p.applyOverrides(ctx, orgID, userID, perm, allowed)
}

// HasOrgPermissionOn reports whether the user may exercise perm on the scoped
// resource (scopeType, scopeID) — for example one region or project. The
// member's organization-wide reach (the membership role plus any additional
// organization-wide assignments) answers every scope; a scoped assignment
// answers only its own scope. Per-member overrides adjust the result exactly
// as in HasOrgPermission: a deny always wins, a grant adds.
//
// An empty scopeType or scopeID degrades to the organization-wide check.
func (p *Plugin) HasOrgPermissionOn(ctx context.Context, userID, orgID string, perm Permission, scopeType, scopeID string) (bool, error) {
	m, err := p.store.GetMember(ctx, userID, orgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	allowed, err := p.orgPermissionDecision(ctx, orgID, userID, m.Role, perm, scopeType, scopeID)
	if err != nil {
		return false, err
	}
	return p.applyOverrides(ctx, orgID, userID, perm, allowed)
}

// orgPermissionDecision resolves perm against the member's organization-wide
// reach — the membership role plus any additional organization-wide
// assignments — and, when scopeType/scopeID identify a resource, against any
// assignment matching that scope. Per-member overrides are applied by the
// caller.
func (p *Plugin) orgPermissionDecision(ctx context.Context, orgID, userID, role string, perm Permission, scopeType, scopeID string) (bool, error) {
	allowed, err := p.roleAllows(ctx, orgID, role, perm)
	if err != nil || allowed {
		return allowed, err
	}
	assignments, err := p.store.ListMemberRoleAssignments(ctx, orgID, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	for _, a := range assignments {
		if a.Scope != nil && (scopeType == "" || scopeID == "" || a.Scope.Type != scopeType || a.Scope.ID != scopeID) {
			continue
		}
		ok, err := p.roleAllows(ctx, orgID, a.Role, perm)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

// applyOverrides resolves a permission decision against a member's override
// set: a deny always wins, a grant adds a permission the role does not carry.
// The decision is returned unchanged when no override matches.
func (p *Plugin) applyOverrides(ctx context.Context, orgID, userID string, perm Permission, allowed bool) (bool, error) {
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
// permission strings are grantable unless the catalog marks them Sensitive.
func (p *Plugin) ungrantablePermission(ctx context.Context, actorID, orgID string, perms []Permission) (Permission, error) {
	held, err := p.orgAuthority(ctx, actorID, orgID)
	if err != nil {
		return "", err
	}
	return p.ungrantableIn(perms, held), nil
}

// orgAuthority resolves actorID's organization-wide effective permission set
// (the membership role plus per-member overrides).
func (p *Plugin) orgAuthority(ctx context.Context, actorID, orgID string) (map[Permission]bool, error) {
	effective, err := p.GetMemberPermissions(ctx, actorID, orgID)
	if err != nil {
		return nil, err
	}
	held := make(map[Permission]bool, len(effective.Permissions))
	for _, perm := range effective.Permissions {
		held[perm] = true
	}
	return held, nil
}

// ungrantableIn returns the first capped permission in perms that held does
// not contain, or "" when every one is grantable. Normalization matches the
// write path: a padded or control-char spelling (" org:delete",
// "org:\x00delete") must be capped by the same rule as the canonical string,
// or it would slip through here and be stored sanitized. Capped permissions
// are the framework set plus catalog entries flagged Sensitive; other
// app-defined permissions are never capped.
func (p *Plugin) ungrantableIn(perms []Permission, held map[Permission]bool) Permission {
	for _, raw := range perms {
		perm := Permission(core.SanitizeString(string(raw), nil))
		if !p.isCapped(perm) {
			continue
		}
		if !held[perm] {
			return perm
		}
	}
	return ""
}

// isCapped reports whether perm is subject to the grant-authority cap:
// framework permissions always are; app-defined permissions only when the
// catalog marks them sensitive.
func (p *Plugin) isCapped(perm Permission) bool {
	if _, ok := frameworkPermissions[perm]; ok {
		return true
	}
	def, ok := p.permissions[perm]
	return ok && def.Sensitive
}

// validatePermissions rejects permission strings outside the configured
// catalog. Without a catalog every non-empty string is accepted (the
// application owns its meaning), so this is a no-op until Config.Permissions
// is set. Empty entries are skipped, matching the storage path.
func (p *Plugin) validatePermissions(perms []Permission) error {
	if len(p.permissions) == 0 {
		return nil
	}
	for _, raw := range perms {
		perm := Permission(core.SanitizeString(string(raw), nil))
		if perm == "" {
			continue
		}
		if _, ok := frameworkPermissions[perm]; ok {
			continue
		}
		if _, ok := p.permissions[perm]; ok {
			continue
		}
		return fmt.Errorf("%w: %q", ErrUnknownPermission, perm)
	}
	return nil
}

// PermissionCatalog returns the framework permissions merged with the
// configured app catalog, sorted by module then permission name, for UIs that
// group and describe permissions. Framework entries are flagged; Sensitive is
// always true for them because they are capped regardless of registration
// metadata.
func (p *Plugin) PermissionCatalog() []PermissionInfo {
	out := make([]PermissionInfo, 0, len(frameworkPermissions)+len(p.permissions))
	for perm := range frameworkPermissions {
		info := PermissionInfo{Permission: perm, Sensitive: true, Framework: true}
		if def, ok := p.permissions[perm]; ok {
			info.Description = def.Description
			info.Module = def.Module
		}
		out = append(out, info)
	}
	for perm, def := range p.permissions {
		if _, ok := frameworkPermissions[perm]; ok {
			continue
		}
		out = append(out, PermissionInfo{
			Permission:  perm,
			Description: def.Description,
			Module:      def.Module,
			Sensitive:   def.Sensitive,
		})
	}
	slices.SortFunc(out, func(a, b PermissionInfo) int {
		if c := strings.Compare(a.Module, b.Module); c != 0 {
			return c
		}
		return strings.Compare(string(a.Permission), string(b.Permission))
	})
	return out
}

// ungrantableOrgRole returns the first permission the org role carries that
// actorID may not grant, or "" when the role is assignable. It applies the
// same authority cap as permission edits to role *assignment*: whoever can
// assign roles must not be able to hand out authority they do not hold
// themselves.
func (p *Plugin) ungrantableOrgRole(ctx context.Context, actorID, orgID, role string) (Permission, error) {
	perms, err := p.rolePermissions(ctx, orgID, role)
	if err != nil {
		return "", err
	}
	return p.ungrantablePermission(ctx, actorID, orgID, perms)
}

// ungrantableTeamRoleOnTeam returns the first permission the team role carries
// that actorID may not grant on the given team. The actor must hold each
// framework permission either organization-wide (org role + overrides) or
// through their own team role on that team, so a team lead keeps managing team
// roles without being able to grant permissions they do not hold.
func (p *Plugin) ungrantableTeamRoleOnTeam(ctx context.Context, actorID, orgID, teamID, role string) (Permission, error) {
	held, err := p.orgAuthority(ctx, actorID, orgID)
	if err != nil {
		return "", err
	}
	tm, err := p.store.GetTeamMember(ctx, teamID, actorID)
	switch {
	case err == nil:
		for _, raw := range p.teamRoles[tm.Role].Permissions {
			held[Permission(core.SanitizeString(string(raw), nil))] = true
		}
	case errors.Is(err, sql.ErrNoRows):
		// Not a team member; organization-wide authority is the whole story.
	default:
		return "", err
	}
	return p.ungrantableIn(p.teamRoles[role].Permissions, held), nil
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

// GetMemberPermissions resolves a member's effective permissions: the
// membership role's permissions plus those of any additional organization-wide
// assignment, with per-member overrides applied (deny wins over grant, grant
// wins over the role). Scoped assignments are not included — they only apply
// at HasOrgPermissionOn. Returns sql.ErrNoRows when the user is not a member.
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

	// Additional organization-wide assignments are part of the member's
	// unscoped reach; scoped assignments stay out of this set.
	assignments, err := p.store.ListMemberRoleAssignments(ctx, orgID, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return MemberPermissions{}, err
	}
	for _, a := range assignments {
		if a.Scope != nil {
			continue
		}
		perms, err := p.rolePermissions(ctx, orgID, a.Role)
		if err != nil {
			return MemberPermissions{}, err
		}
		for _, perm := range perms {
			effective[perm] = true
		}
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

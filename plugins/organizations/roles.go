package organizations

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/theinventorylib/aegis/v2/core"
	orgtypes "github.com/theinventorylib/aegis/v2/plugins/organizations/types"
)

// Role-management errors.
var (
	// ErrRoleReserved means the name belongs to a compiled (built-in or
	// Config.OrgRoles) role, which cannot be edited through the API.
	ErrRoleReserved = errors.New("role name is reserved")

	// ErrRoleExists means a custom role with that name already exists.
	ErrRoleExists = errors.New("role already exists")

	// ErrRoleInUse means members still hold the role, so it cannot be deleted.
	ErrRoleInUse = errors.New("role is assigned to members")
)

// RoleInfo describes a role in an organization's catalog.
type RoleInfo struct {
	Name        string       `json:"name"`
	Permissions []Permission `json:"permissions"`
	Custom      bool         `json:"custom"`
}

// rolePermissions resolves a role name to its permission list. Compiled
// definitions (built-ins + Config.OrgRoles) win over persisted custom roles, so
// an organization can never shadow a built-in role.
func (p *Plugin) rolePermissions(ctx context.Context, orgID, role string) ([]Permission, error) {
	if def, ok := p.orgRoles[role]; ok {
		return def.Permissions, nil
	}
	custom, err := p.store.GetOrganizationRole(ctx, orgID, role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	perms := make([]Permission, 0, len(custom.Permissions))
	for _, perm := range custom.Permissions {
		perms = append(perms, Permission(perm))
	}
	return perms, nil
}

// roleAllows reports whether a role grants perm, consulting persisted custom
// roles when the name is not compiled in.
func (p *Plugin) roleAllows(ctx context.Context, orgID, role string, perm Permission) (bool, error) {
	perms, err := p.rolePermissions(ctx, orgID, role)
	if err != nil {
		return false, err
	}
	return slices.Contains(perms, perm), nil
}

// IsCompiledRole reports whether name is a built-in or Config.OrgRoles role.
// Compiled roles cannot be edited or deleted through the API.
func (p *Plugin) IsCompiledRole(name string) bool {
	_, ok := p.orgRoles[name]
	return ok
}

// ListRoles returns the compiled role catalog merged with the organization's
// persisted custom roles. Compiled definitions win on name collisions.
func (p *Plugin) ListRoles(ctx context.Context, orgID string) ([]RoleInfo, error) {
	roles := make([]RoleInfo, 0, len(p.orgRoles))
	seen := make(map[string]bool, len(p.orgRoles))
	for name, def := range p.orgRoles {
		roles = append(roles, RoleInfo{Name: name, Permissions: def.Permissions, Custom: false})
		seen[name] = true
	}

	custom, err := p.store.ListOrganizationRoles(ctx, orgID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	for _, role := range custom {
		if seen[role.Name] {
			continue
		}
		perms := make([]Permission, 0, len(role.Permissions))
		for _, perm := range role.Permissions {
			perms = append(perms, Permission(perm))
		}
		roles = append(roles, RoleInfo{Name: role.Name, Permissions: perms, Custom: true})
	}

	slices.SortFunc(roles, func(a, b RoleInfo) int { return strings.Compare(a.Name, b.Name) })
	return roles, nil
}

// CreateRole persists a custom role for the organization. The name must not
// collide with a compiled role or an existing custom role.
func (p *Plugin) CreateRole(ctx context.Context, orgID, name string, permissions []Permission) (RoleInfo, error) {
	name = core.SanitizeString(name, nil)
	if p.IsCompiledRole(name) {
		return RoleInfo{}, ErrRoleReserved
	}
	if _, err := p.store.GetOrganizationRole(ctx, orgID, name); err == nil {
		return RoleInfo{}, ErrRoleExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return RoleInfo{}, err
	}

	now := time.Now()
	role := orgtypes.OrganizationRole{
		ID:             core.GenerateID(),
		OrganizationID: orgID,
		Name:           name,
		Permissions:    permissionStrings(permissions),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := p.store.CreateOrganizationRole(ctx, role); err != nil {
		return RoleInfo{}, err
	}
	return RoleInfo{Name: name, Permissions: dedupePermissions(permissions), Custom: true}, nil
}

// UpdateRole replaces a custom role's permissions. Compiled roles are refused;
// sql.ErrNoRows means the role does not exist for this organization.
func (p *Plugin) UpdateRole(ctx context.Context, orgID, name string, permissions []Permission) (RoleInfo, error) {
	if p.IsCompiledRole(name) {
		return RoleInfo{}, ErrRoleReserved
	}
	role, err := p.store.GetOrganizationRole(ctx, orgID, name)
	if err != nil {
		return RoleInfo{}, err
	}
	role.Permissions = permissionStrings(permissions)
	role.UpdatedAt = time.Now()
	if err := p.store.UpdateOrganizationRole(ctx, role); err != nil {
		return RoleInfo{}, err
	}
	return RoleInfo{Name: name, Permissions: dedupePermissions(permissions), Custom: true}, nil
}

// DeleteRole removes a custom role. Roles still assigned to members are
// refused so members do not silently lose their permissions.
func (p *Plugin) DeleteRole(ctx context.Context, orgID, name string) error {
	if p.IsCompiledRole(name) {
		return ErrRoleReserved
	}
	if _, err := p.store.GetOrganizationRole(ctx, orgID, name); err != nil {
		return err
	}
	count, err := p.store.CountOrganizationMembersWithRole(ctx, orgID, name)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrRoleInUse
	}
	return p.store.DeleteOrganizationRole(ctx, orgID, name)
}

// permissionsFromStrings converts request strings to Permission values.
func permissionsFromStrings(perms []string) []Permission {
	out := make([]Permission, 0, len(perms))
	for _, perm := range perms {
		out = append(out, Permission(perm))
	}
	return out
}

// permissionStrings sanitizes and deduplicates a permission list for storage.
func permissionStrings(perms []Permission) []string {
	out := make([]string, 0, len(perms))
	seen := make(map[Permission]bool, len(perms))
	for _, perm := range perms {
		perm = Permission(core.SanitizeString(string(perm), nil))
		if perm == "" || seen[perm] {
			continue
		}
		seen[perm] = true
		out = append(out, string(perm))
	}
	return out
}

// dedupePermissions mirrors permissionStrings for API responses.
func dedupePermissions(perms []Permission) []Permission {
	out := make([]Permission, 0, len(perms))
	seen := make(map[Permission]bool, len(perms))
	for _, perm := range perms {
		if perm == "" || seen[perm] {
			continue
		}
		seen[perm] = true
		out = append(out, perm)
	}
	return out
}

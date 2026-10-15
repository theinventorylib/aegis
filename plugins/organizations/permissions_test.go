package organizations

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	orgtypes "github.com/theinventorylib/aegis/v2/plugins/organizations/types"
)

// fakeStore embeds the interface so only the methods used by a test need to be
// implemented.
type fakeStore struct {
	orgtypes.OrganizationStore
	memberOrgRole   string
	teamRole        string
	overrides       []orgtypes.MemberPermissionOverride
	customRoles     map[string]orgtypes.OrganizationRole
	roleMemberCount int
	assignments     []orgtypes.RoleAssignment
	assignmentsErr  error
}

func (f *fakeStore) GetMember(_ context.Context, userID, orgID string) (orgtypes.Member, error) {
	if f.memberOrgRole == "" {
		return orgtypes.Member{}, sql.ErrNoRows
	}
	return orgtypes.Member{ID: "m1", UserID: userID, OrganizationID: orgID, Role: f.memberOrgRole}, nil
}

func (f *fakeStore) UpdateMemberRole(_ context.Context, _, _, role string, _ time.Time) error {
	f.memberOrgRole = role
	return nil
}

func (f *fakeStore) ListMemberRoleAssignments(_ context.Context, _, _ string) ([]orgtypes.RoleAssignment, error) {
	if f.assignmentsErr != nil {
		return nil, f.assignmentsErr
	}
	return f.assignments, nil
}

func (f *fakeStore) CreateMemberRoleAssignment(_ context.Context, id, orgID, userID, role, scopeType, scopeID string, createdAt, updatedAt time.Time) error {
	var scope *orgtypes.Scope
	if scopeType != "" || scopeID != "" {
		scope = &orgtypes.Scope{Type: scopeType, ID: scopeID}
	}
	f.assignments = append(f.assignments, orgtypes.RoleAssignment{
		ID: id, OrganizationID: orgID, UserID: userID, Role: role,
		Scope:     scope,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
	return nil
}

func (f *fakeStore) DeleteMemberRoleAssignment(_ context.Context, _, _, id string) error {
	for i, a := range f.assignments {
		if a.ID == id {
			f.assignments = append(f.assignments[:i], f.assignments[i+1:]...)
			return nil
		}
	}
	return sql.ErrNoRows
}

func (f *fakeStore) GetTeamMember(_ context.Context, _, _ string) (orgtypes.TeamMember, error) {
	if f.teamRole == "" {
		return orgtypes.TeamMember{}, sql.ErrNoRows
	}
	return orgtypes.TeamMember{Role: f.teamRole}, nil
}

func (f *fakeStore) ListMemberPermissionOverrides(_ context.Context, _, _ string) ([]orgtypes.MemberPermissionOverride, error) {
	return f.overrides, nil
}

func (f *fakeStore) CreateMemberPermissionOverride(_ context.Context, o orgtypes.MemberPermissionOverride) error {
	f.overrides = append(f.overrides, o)
	return nil
}

func (f *fakeStore) DeleteMemberPermissionOverrides(_ context.Context, _, _ string) error {
	f.overrides = nil
	return nil
}

func (f *fakeStore) ListOrganizationRoles(_ context.Context, _ string) ([]orgtypes.OrganizationRole, error) {
	out := make([]orgtypes.OrganizationRole, 0, len(f.customRoles))
	for _, r := range f.customRoles {
		out = append(out, r)
	}
	return out, nil
}

func (f *fakeStore) GetOrganizationRole(_ context.Context, _, name string) (orgtypes.OrganizationRole, error) {
	if r, ok := f.customRoles[name]; ok {
		return r, nil
	}
	return orgtypes.OrganizationRole{}, sql.ErrNoRows
}

func (f *fakeStore) CreateOrganizationRole(_ context.Context, role orgtypes.OrganizationRole) error {
	if f.customRoles == nil {
		f.customRoles = map[string]orgtypes.OrganizationRole{}
	}
	f.customRoles[role.Name] = role
	return nil
}

func (f *fakeStore) UpdateOrganizationRole(_ context.Context, role orgtypes.OrganizationRole) error {
	if _, ok := f.customRoles[role.Name]; !ok {
		return sql.ErrNoRows
	}
	f.customRoles[role.Name] = role
	return nil
}

func (f *fakeStore) DeleteOrganizationRole(_ context.Context, _, name string) error {
	delete(f.customRoles, name)
	return nil
}

func (f *fakeStore) CountOrganizationMembersWithRole(_ context.Context, _, _ string) (int, error) {
	return f.roleMemberCount, nil
}

func TestRoleDefinition_Allows(t *testing.T) {
	def := RoleDefinition{Permissions: []Permission{PermOrgView, PermMemberView}}
	if !def.Allows(PermOrgView) {
		t.Error("expected PermOrgView to be granted")
	}
	if def.Allows(PermOrgManage) {
		t.Error("expected PermOrgManage to be denied")
	}
	if (RoleDefinition{}).Allows(PermOrgView) {
		t.Error("zero definition must grant nothing")
	}
}

func TestNewResolvesBuiltInAndCustomRoles(t *testing.T) {
	p := New(&Config{
		OrgRoles: map[string]RoleDefinition{
			"billing": {Permissions: []Permission{PermOrgView, PermMemberView}},
		},
		TeamRoles: map[string]RoleDefinition{
			"reviewer": {Permissions: []Permission{PermTeamView}},
		},
	}, nil)

	orgRoles := p.OrgRoles()
	if !orgRoles["billing"].Allows(PermOrgView) {
		t.Error("custom org role should grant its configured permission")
	}
	if orgRoles["billing"].Allows(PermOrgManage) {
		t.Error("custom org role must not grant unconfigured permissions")
	}
	// Built-ins survive the merge.
	if !orgRoles[orgtypes.RoleOwner].Allows(PermOrgDelete) {
		t.Error("owner should still grant PermOrgDelete")
	}
	if !orgRoles[orgtypes.RoleAdmin].Allows(PermMemberManage) {
		t.Error("admin should still grant PermMemberManage")
	}
	if orgRoles[orgtypes.RoleMember].Allows(PermMemberManage) {
		t.Error("member must not grant PermMemberManage")
	}
	if _, ok := p.TeamRoles()["reviewer"]; !ok {
		t.Error("custom team role should be present")
	}
}

func TestNewReplacesBuiltInRole(t *testing.T) {
	p := New(&Config{
		OrgRoles: map[string]RoleDefinition{
			// Downgrade member to nothing.
			orgtypes.RoleMember: {Permissions: nil},
		},
	}, nil)

	if p.OrgRoles()[orgtypes.RoleMember].Allows(PermOrgView) {
		t.Error("overriding member should replace its permissions")
	}
}

func TestNewNilConfig(t *testing.T) {
	p := New(nil, nil)
	if len(p.OrgRoles()) == 0 || len(p.TeamRoles()) == 0 {
		t.Fatal("nil config must still produce built-in roles")
	}
}

func TestOrgMemberRolesExcludeOwner(t *testing.T) {
	p := New(&Config{
		OrgRoles: map[string]RoleDefinition{
			"billing": {Permissions: []Permission{PermOrgView}},
		},
	}, nil)

	for _, r := range p.orgMemberRoles() {
		if r == orgtypes.RoleOwner {
			t.Fatal("owner must not be assignable through member endpoints")
		}
	}
	if len(p.orgMemberRoles()) != 3 { // admin, member, billing
		t.Errorf("orgMemberRoles() = %v, want 3 entries", p.orgMemberRoles())
	}
}

func TestHasOrgPermission(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		role string
		perm Permission
		want bool
	}{
		{"owner can delete", orgtypes.RoleOwner, PermOrgDelete, true},
		{"admin cannot delete", orgtypes.RoleAdmin, PermOrgDelete, false},
		{"admin can manage members", orgtypes.RoleAdmin, PermMemberManage, true},
		{"member can view", orgtypes.RoleMember, PermOrgView, true},
		{"member cannot manage", orgtypes.RoleMember, PermMemberManage, false},
		{"unknown role has no perms", "ghost", PermOrgView, false},
		{"non-member has no perms", "", PermOrgView, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New(nil, &fakeStore{memberOrgRole: tc.role})
			got, err := p.HasOrgPermission(ctx, "u1", "o1", tc.perm)
			if err != nil {
				t.Fatalf("HasOrgPermission: %v", err)
			}
			if got != tc.want {
				t.Errorf("HasOrgPermission(%s) = %v, want %v", tc.perm, got, tc.want)
			}
		})
	}
}

func TestHasTeamPermission(t *testing.T) {
	ctx := context.Background()
	p := New(nil, &fakeStore{teamRole: orgtypes.RoleTeamLead})
	if ok, _ := p.HasTeamPermission(ctx, "u1", "t1", PermTeamMemberManage); !ok {
		t.Error("team lead should manage team members")
	}
	if ok, _ := p.HasTeamPermission(ctx, "u1", "t1", PermTeamManage); ok {
		t.Error("team lead should not manage the team itself")
	}
}

func TestHasOrgPermissionWithOverrides(t *testing.T) {
	ctx := context.Background()
	grant := func(perm Permission) []orgtypes.MemberPermissionOverride {
		return []orgtypes.MemberPermissionOverride{{Permission: string(perm), Effect: orgtypes.PermissionEffectGrant}}
	}
	deny := func(perm Permission) []orgtypes.MemberPermissionOverride {
		return []orgtypes.MemberPermissionOverride{{Permission: string(perm), Effect: orgtypes.PermissionEffectDeny}}
	}

	cases := []struct {
		name      string
		role      string
		overrides []orgtypes.MemberPermissionOverride
		perm      Permission
		want      bool
	}{
		{"grant adds a permission", orgtypes.RoleMember, grant(PermMemberManage), PermMemberManage, true},
		{"deny removes a role permission", orgtypes.RoleAdmin, deny(PermMemberManage), PermMemberManage, false},
		{"deny leaves other permissions alone", orgtypes.RoleAdmin, deny(PermMemberManage), PermOrgView, true},
		{"grant for another permission is ignored", orgtypes.RoleMember, grant(PermOrgManage), PermMemberManage, false},
		{"overrides never apply to non-members", "", grant(PermMemberManage), PermMemberManage, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New(nil, &fakeStore{memberOrgRole: tc.role, overrides: tc.overrides})
			got, err := p.HasOrgPermission(ctx, "u1", "o1", tc.perm)
			if err != nil {
				t.Fatalf("HasOrgPermission: %v", err)
			}
			if got != tc.want {
				t.Errorf("HasOrgPermission(%s) = %v, want %v", tc.perm, got, tc.want)
			}
		})
	}
}

func TestGetMemberPermissionsResolvesOverrides(t *testing.T) {
	ctx := context.Background()
	p := New(nil, &fakeStore{
		memberOrgRole: orgtypes.RoleAdmin,
		overrides: []orgtypes.MemberPermissionOverride{
			{Permission: string(PermMemberManage), Effect: orgtypes.PermissionEffectDeny},
			{Permission: "view_giving", Effect: orgtypes.PermissionEffectGrant},
		},
	})

	perms, err := p.GetMemberPermissions(ctx, "u1", "o1")
	if err != nil {
		t.Fatalf("GetMemberPermissions: %v", err)
	}
	if perms.Role != orgtypes.RoleAdmin {
		t.Errorf("role = %q, want admin", perms.Role)
	}
	if len(perms.Overrides) != 2 {
		t.Errorf("overrides = %d, want 2", len(perms.Overrides))
	}
	has := func(perm Permission) bool {
		for _, p := range perms.Permissions {
			if p == perm {
				return true
			}
		}
		return false
	}
	if has(PermMemberManage) {
		t.Error("denied permission should be removed from the effective set")
	}
	if !has(PermOrgView) {
		t.Error("untouched role permission should remain")
	}
	if !has("view_giving") {
		t.Error("granted app permission should be present")
	}

	if _, err := New(nil, &fakeStore{}).GetMemberPermissions(ctx, "u1", "o1"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("non-member: got %v, want sql.ErrNoRows", err)
	}
}

func TestCustomRoleGrantsPermissions(t *testing.T) {
	ctx := context.Background()
	store := &fakeStore{
		memberOrgRole: "billing",
		customRoles: map[string]orgtypes.OrganizationRole{
			"billing": {Name: "billing", Permissions: []string{string(PermOrgView), "view_giving"}},
		},
	}
	p := New(nil, store)

	if ok, err := p.HasOrgPermission(ctx, "u1", "o1", PermOrgView); err != nil || !ok {
		t.Errorf("custom role should grant org:view (ok=%v err=%v)", ok, err)
	}
	if ok, err := p.HasOrgPermission(ctx, "u1", "o1", "view_giving"); err != nil || !ok {
		t.Errorf("custom role should grant app permission (ok=%v err=%v)", ok, err)
	}
	if ok, _ := p.HasOrgPermission(ctx, "u1", "o1", PermOrgManage); ok {
		t.Error("custom role must not grant permissions it does not list")
	}
}

func TestCompiledRoleWinsOverCustomRole(t *testing.T) {
	ctx := context.Background()
	store := &fakeStore{
		memberOrgRole: orgtypes.RoleAdmin,
		customRoles: map[string]orgtypes.OrganizationRole{
			"admin": {Name: "admin", Permissions: []string{string(PermOrgDelete)}},
		},
	}
	p := New(nil, store)
	if ok, _ := p.HasOrgPermission(ctx, "u1", "o1", PermOrgDelete); ok {
		t.Error("a custom role must not shadow the built-in admin role")
	}
}

func TestCreateUpdateDeleteRole(t *testing.T) {
	ctx := context.Background()
	store := &fakeStore{memberOrgRole: orgtypes.RoleOwner}
	p := New(nil, store)

	if _, err := p.CreateRole(ctx, "o1", "admin", nil); !errors.Is(err, ErrRoleReserved) {
		t.Fatalf("compiled name: got %v, want ErrRoleReserved", err)
	}
	if _, err := p.CreateRole(ctx, "o1", "billing", []Permission{PermOrgView}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := p.CreateRole(ctx, "o1", "billing", nil); !errors.Is(err, ErrRoleExists) {
		t.Fatalf("duplicate: got %v, want ErrRoleExists", err)
	}

	info, err := p.UpdateRole(ctx, "o1", "billing", []Permission{PermOrgView, PermMemberView})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(info.Permissions) != 2 {
		t.Errorf("updated permissions = %v, want 2 entries", info.Permissions)
	}
	if _, err := p.UpdateRole(ctx, "o1", "ghost", nil); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unknown role: got %v, want sql.ErrNoRows", err)
	}

	store.roleMemberCount = 1
	if err := p.DeleteRole(ctx, "o1", "billing"); !errors.Is(err, ErrRoleInUse) {
		t.Fatalf("in-use role: got %v, want ErrRoleInUse", err)
	}
	store.roleMemberCount = 0
	if err := p.DeleteRole(ctx, "o1", "billing"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := p.DeleteRole(ctx, "o1", "admin"); !errors.Is(err, ErrRoleReserved) {
		t.Fatalf("compiled delete: got %v, want ErrRoleReserved", err)
	}
}

func TestListRolesMergesCatalog(t *testing.T) {
	ctx := context.Background()
	p := New(nil, &fakeStore{
		customRoles: map[string]orgtypes.OrganizationRole{
			"billing": {Name: "billing", Permissions: []string{string(PermOrgView)}},
			"admin":   {Name: "admin", Permissions: []string{string(PermOrgDelete)}},
		},
	})
	roles, err := p.ListRoles(ctx, "o1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	byName := map[string]RoleInfo{}
	for _, r := range roles {
		byName[r.Name] = r
	}
	if r, ok := byName["billing"]; !ok || !r.Custom {
		t.Error("custom role should be listed as custom")
	}
	if r, ok := byName["admin"]; !ok || r.Custom {
		t.Error("compiled role must win and be listed as non-custom")
	}
	if r := byName[orgtypes.RoleOwner]; len(r.Permissions) == 0 {
		t.Error("compiled owner role should be listed with its permissions")
	}
}

func TestAssignableOrgRolesIncludesCustom(t *testing.T) {
	ctx := context.Background()
	p := New(nil, &fakeStore{
		customRoles: map[string]orgtypes.OrganizationRole{
			"billing": {Name: "billing", Permissions: []string{string(PermOrgView)}},
			"owner":   {Name: "owner", Permissions: nil},
		},
	})

	req := AddOrganizationMemberRequest{UserID: "u1", Role: "billing"}
	if err := p.ValidateAddMember(ctx, "o1", req); err != nil {
		t.Fatalf("custom role should be assignable: %v", err)
	}
	req.Role = "ghost"
	if err := p.ValidateAddMember(ctx, "o1", req); err == nil {
		t.Fatal("unknown role must be rejected")
	}
	req.Role = orgtypes.RoleOwner
	if err := p.ValidateAddMember(ctx, "o1", req); err == nil {
		t.Fatal("owner must never be assignable through member endpoints")
	}
}

func TestSetMemberPermissionOverrides(t *testing.T) {
	ctx := context.Background()
	store := &fakeStore{memberOrgRole: orgtypes.RoleAdmin}
	p := New(nil, store)

	err := p.SetMemberPermissionOverrides(ctx, "o1", "u1", []orgtypes.MemberPermissionOverride{
		{Permission: string(PermMemberManage), Effect: orgtypes.PermissionEffectDeny},
		{Permission: "view_giving", Effect: orgtypes.PermissionEffectGrant},
	})
	if err != nil {
		t.Fatalf("set overrides: %v", err)
	}
	if len(store.overrides) != 2 {
		t.Fatalf("stored %d overrides, want 2", len(store.overrides))
	}
	for _, o := range store.overrides {
		if o.ID == "" || o.OrganizationID != "o1" || o.UserID != "u1" || o.CreatedAt.IsZero() || o.UpdatedAt.IsZero() {
			t.Errorf("override not filled in: %+v", o)
		}
	}

	// Replacement clears the previous set.
	if err := p.SetMemberPermissionOverrides(ctx, "o1", "u1", nil); err != nil {
		t.Fatalf("clear overrides: %v", err)
	}
	if len(store.overrides) != 0 {
		t.Fatalf("stored %d overrides after clear, want 0", len(store.overrides))
	}

	// Duplicates and unknown effects are rejected before any write.
	if err := p.SetMemberPermissionOverrides(ctx, "o1", "u1", []orgtypes.MemberPermissionOverride{
		{Permission: "view_giving", Effect: orgtypes.PermissionEffectGrant},
		{Permission: "view_giving", Effect: orgtypes.PermissionEffectDeny},
	}); err == nil {
		t.Fatal("duplicate permission must be rejected")
	}
	if err := p.SetMemberPermissionOverrides(ctx, "o1", "u1", []orgtypes.MemberPermissionOverride{
		{Permission: "view_giving", Effect: "sideways"},
	}); err == nil {
		t.Fatal("unknown effect must be rejected")
	}

	// Non-members cannot carry overrides.
	nonMember := New(nil, &fakeStore{})
	if err := nonMember.SetMemberPermissionOverrides(ctx, "o1", "u1", nil); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("non-member: got %v, want sql.ErrNoRows", err)
	}
}

func TestUngrantableRole(t *testing.T) {
	ctx := context.Background()
	p := New(&Config{OrgRoles: map[string]RoleDefinition{
		"superadmin": {Permissions: []Permission{PermOrgView, PermOrgDelete}},
	}}, &fakeStore{memberOrgRole: orgtypes.RoleAdmin})

	// An admin cannot assign a role carrying org:delete...
	if bad, err := p.ungrantableOrgRole(ctx, "u1", "o1", "superadmin"); err != nil || bad != PermOrgDelete {
		t.Fatalf("org role carrying org:delete: bad=%q err=%v, want %q", bad, err, PermOrgDelete)
	}
	// ...but roles within the actor's own authority are assignable.
	if bad, err := p.ungrantableOrgRole(ctx, "u1", "o1", orgtypes.RoleAdmin); err != nil || bad != "" {
		t.Fatalf("role within authority: bad=%q err=%v, want grantable", bad, err)
	}

	// An org admin holds the team permissions, so team roles are grantable
	// even without a team membership.
	if bad, err := p.ungrantableTeamRoleOnTeam(ctx, "u1", "o1", "team-1", orgtypes.RoleTeamLead); err != nil || bad != "" {
		t.Fatalf("team role within org authority: bad=%q err=%v, want grantable", bad, err)
	}

	// A team lead's authority is their org role plus their team role: they can
	// keep granting roles within it...
	lead := New(&Config{TeamRoles: map[string]RoleDefinition{
		"co-lead": {Permissions: []Permission{PermTeamManage}},
	}}, &fakeStore{memberOrgRole: orgtypes.RoleMember, teamRole: orgtypes.RoleTeamLead})
	if bad, err := lead.ungrantableTeamRoleOnTeam(ctx, "u1", "o1", "team-1", orgtypes.RoleTeamLead); err != nil || bad != "" {
		t.Fatalf("team lead granting lead: bad=%q err=%v, want grantable", bad, err)
	}
	// ...but not a team role carrying a permission they do not hold.
	if bad, err := lead.ungrantableTeamRoleOnTeam(ctx, "u1", "o1", "team-1", "co-lead"); err != nil || bad != PermTeamManage {
		t.Fatalf("team lead granting co-lead: bad=%q err=%v, want %q", bad, err, PermTeamManage)
	}

	// The owner holds org:delete, so the same role is grantable by them.
	owner := New(nil, &fakeStore{memberOrgRole: orgtypes.RoleOwner})
	if bad, err := owner.ungrantableOrgRole(ctx, "u1", "o1", "superadmin"); err != nil || bad != "" {
		t.Fatalf("owner assigning superadmin: bad=%q err=%v, want grantable", bad, err)
	}
}

func TestHasOrgPermissionOn(t *testing.T) {
	ctx := context.Background()
	grant := func(perm Permission) []orgtypes.MemberPermissionOverride {
		return []orgtypes.MemberPermissionOverride{{Permission: string(perm), Effect: orgtypes.PermissionEffectGrant}}
	}
	deny := func(perm Permission) []orgtypes.MemberPermissionOverride {
		return []orgtypes.MemberPermissionOverride{{Permission: string(perm), Effect: orgtypes.PermissionEffectDeny}}
	}
	scoped := func(role, typ, id string) []orgtypes.RoleAssignment {
		return []orgtypes.RoleAssignment{{ID: "a1", Role: role, Scope: &orgtypes.Scope{Type: typ, ID: id}}}
	}
	unscoped := func(role string) []orgtypes.RoleAssignment {
		return []orgtypes.RoleAssignment{{ID: "a2", Role: role}}
	}

	cases := []struct {
		name           string
		role           string
		assignments    []orgtypes.RoleAssignment
		assignmentsErr error
		overrides      []orgtypes.MemberPermissionOverride
		scopeType      string
		scopeID        string
		perm           Permission
		want           bool
	}{
		{name: "unscoped role answers every scope", role: orgtypes.RoleAdmin, scopeType: "campus", scopeID: "c1", perm: PermMemberManage, want: true},
		{name: "scoped assignment answers its scope", role: orgtypes.RoleMember, assignments: scoped("campus-pastor", "campus", "c1"), scopeType: "campus", scopeID: "c1", perm: PermMemberManage, want: true},
		{name: "scoped assignment ignored outside its scope", role: orgtypes.RoleMember, assignments: scoped("campus-pastor", "campus", "c1"), scopeType: "campus", scopeID: "c2", perm: PermMemberManage, want: false},
		{name: "scope type must match", role: orgtypes.RoleMember, assignments: scoped("campus-pastor", "campus", "c1"), scopeType: "service", scopeID: "c1", perm: PermMemberManage, want: false},
		{name: "scoped role must grant the permission", role: orgtypes.RoleMember, assignments: scoped("campus-pastor", "campus", "c1"), scopeType: "campus", scopeID: "c1", perm: PermOrgDelete, want: false},
		{name: "deny override beats a scoped grant", role: orgtypes.RoleMember, assignments: scoped("campus-pastor", "campus", "c1"), overrides: deny(PermMemberManage), scopeType: "campus", scopeID: "c1", perm: PermMemberManage, want: false},
		{name: "grant override applies", role: orgtypes.RoleMember, overrides: grant(PermMemberManage), scopeType: "campus", scopeID: "c1", perm: PermMemberManage, want: true},
		{name: "scoped assignments do not answer the unscoped check", role: orgtypes.RoleMember, assignments: scoped("campus-pastor", "campus", "c1"), perm: PermMemberManage, want: false},
		{name: "additional org-wide assignment answers the unscoped check", role: orgtypes.RoleMember, assignments: unscoped("campus-pastor"), perm: PermMemberManage, want: true},
		{name: "additional org-wide assignment answers any scope", role: orgtypes.RoleMember, assignments: unscoped("campus-pastor"), scopeType: "campus", scopeID: "c1", perm: PermMemberManage, want: true},
		{name: "non-member has no reach", assignments: scoped("campus-pastor", "campus", "c1"), scopeType: "campus", scopeID: "c1", perm: PermMemberManage, want: false},
		{name: "no-rows assignment listing means no scoped grant", role: orgtypes.RoleMember, assignmentsErr: sql.ErrNoRows, scopeType: "campus", scopeID: "c1", perm: PermMemberManage, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New(&Config{OrgRoles: map[string]RoleDefinition{
				"campus-pastor": {Permissions: []Permission{PermOrgView, PermMemberManage}},
			}}, &fakeStore{memberOrgRole: tc.role, assignments: tc.assignments, assignmentsErr: tc.assignmentsErr, overrides: tc.overrides})
			got, err := p.HasOrgPermissionOn(ctx, "u1", "o1", tc.perm, tc.scopeType, tc.scopeID)
			if err != nil {
				t.Fatalf("HasOrgPermissionOn: %v", err)
			}
			if got != tc.want {
				t.Errorf("HasOrgPermissionOn(%s @ %s/%s) = %v, want %v", tc.perm, tc.scopeType, tc.scopeID, got, tc.want)
			}
		})
	}
}

func TestRoleAssignmentAPI(t *testing.T) {
	ctx := context.Background()
	store := &fakeStore{memberOrgRole: orgtypes.RoleMember}
	p := New(nil, store)

	// A nil scope replaces the organization-wide role.
	if err := p.AssignRole(ctx, "o1", "u1", orgtypes.RoleAdmin, nil); err != nil {
		t.Fatalf("assign unscoped: %v", err)
	}
	if store.memberOrgRole != orgtypes.RoleAdmin {
		t.Fatalf("unscoped role = %q, want admin", store.memberOrgRole)
	}

	// A scoped assignment is stored with its opaque scope.
	if err := p.AssignRole(ctx, "o1", "u1", "campus-pastor", &orgtypes.Scope{Type: "campus", ID: "c1"}); err != nil {
		t.Fatalf("assign scoped: %v", err)
	}
	if len(store.assignments) != 1 {
		t.Fatalf("assignments = %d, want 1", len(store.assignments))
	}
	a := store.assignments[0]
	if a.ID == "" || a.OrganizationID != "o1" || a.UserID != "u1" || a.Role != "campus-pastor" ||
		a.Scope == nil || a.Scope.Type != "campus" || a.Scope.ID != "c1" || a.CreatedAt.IsZero() || a.UpdatedAt.IsZero() {
		t.Fatalf("assignment not filled in: %+v", a)
	}

	// Assigning the same role and scope again is idempotent.
	if err := p.AssignRole(ctx, "o1", "u1", "campus-pastor", &orgtypes.Scope{Type: "campus", ID: "c1"}); err != nil {
		t.Fatalf("re-assign scoped: %v", err)
	}
	if len(store.assignments) != 1 {
		t.Fatalf("assignments after re-assign = %d, want 1", len(store.assignments))
	}

	// List returns the organization-wide assignment first, then scoped ones.
	list, err := p.ListRoleAssignments(ctx, "o1", "u1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 || list[0].Scope != nil || list[0].Role != orgtypes.RoleAdmin || list[0].ID != "m1" || !list[0].Primary || list[1].Primary || list[1].ID != a.ID {
		t.Fatalf("list = %+v, want [primary admin, scoped %s]", list, a.ID)
	}

	// Remove drops the scoped assignment.
	if err := p.RemoveRoleAssignment(ctx, "o1", "u1", a.ID); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if len(store.assignments) != 0 {
		t.Fatalf("assignments after remove = %d, want 0", len(store.assignments))
	}
}

func TestRoleAssignmentAPIValidation(t *testing.T) {
	ctx := context.Background()
	p := New(nil, &fakeStore{memberOrgRole: orgtypes.RoleMember})

	// A scope missing either half is rejected before anything is written.
	if err := p.AssignRole(ctx, "o1", "u1", "campus-pastor", &orgtypes.Scope{Type: "campus"}); err == nil {
		t.Fatal("scope without an id must be rejected")
	}
	// Scopes longer than the cross-dialect limit are rejected too.
	if err := p.AssignRole(ctx, "o1", "u1", "campus-pastor", &orgtypes.Scope{Type: "campus", ID: strings.Repeat("x", maxScopeLen+1)}); err == nil {
		t.Fatal("over-long scope id must be rejected")
	}

	// Removing an unknown (or primary) ID surfaces sql.ErrNoRows.
	if err := p.RemoveRoleAssignment(ctx, "o1", "u1", "missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("remove unknown: got %v, want sql.ErrNoRows", err)
	}

	// Assignments require membership, on both paths.
	if err := New(nil, &fakeStore{}).AssignRole(ctx, "o1", "u1", "member", &orgtypes.Scope{Type: "campus", ID: "c1"}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("non-member scoped assign: got %v, want sql.ErrNoRows", err)
	}
	if err := New(nil, &fakeStore{}).AssignRole(ctx, "o1", "u1", orgtypes.RoleMember, nil); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("non-member unscoped assign: got %v, want sql.ErrNoRows", err)
	}
}

func TestAddRoleAssignmentAndUnscopedReach(t *testing.T) {
	ctx := context.Background()
	store := &fakeStore{memberOrgRole: orgtypes.RoleMember}
	p := New(&Config{OrgRoles: map[string]RoleDefinition{
		"finance-admin": {Permissions: []Permission{PermOrgView, PermOrgManage}},
	}}, store)

	// Adding the membership's current primary role organization-wide is a
	// no-op: the member already holds it.
	if err := p.AddRoleAssignment(ctx, "o1", "u1", orgtypes.RoleMember, nil); err != nil {
		t.Fatalf("add primary role: %v", err)
	}
	if len(store.assignments) != 0 {
		t.Fatalf("assignments = %d, want 0 (primary role already grants it)", len(store.assignments))
	}

	// An additional organization-wide assignment...
	if err := p.AddRoleAssignment(ctx, "o1", "u1", "finance-admin", nil); err != nil {
		t.Fatalf("add unscoped: %v", err)
	}
	if len(store.assignments) != 1 || store.assignments[0].Scope != nil {
		t.Fatalf("assignments = %+v, want one with nil scope", store.assignments)
	}
	// ...is idempotent...
	if err := p.AddRoleAssignment(ctx, "o1", "u1", "finance-admin", nil); err != nil {
		t.Fatalf("re-add unscoped: %v", err)
	}
	if len(store.assignments) != 1 {
		t.Fatalf("assignments after re-add = %d, want 1", len(store.assignments))
	}
	// ...a scoped add is the same thing with a scope...
	if err := p.AddRoleAssignment(ctx, "o1", "u1", "campus-pastor", &orgtypes.Scope{Type: "campus", ID: "c1"}); err != nil {
		t.Fatalf("add scoped: %v", err)
	}
	if len(store.assignments) != 2 || store.assignments[1].Scope == nil {
		t.Fatalf("assignments = %+v, want a second scoped entry", store.assignments)
	}

	// ...entitles the member organization-wide even though the primary role
	// does not...
	ok, err := p.HasOrgPermission(ctx, "u1", "o1", PermOrgManage)
	if err != nil || !ok {
		t.Fatalf("HasOrgPermission(org:manage) = %v, %v; want true", ok, err)
	}
	// ...answers every scope...
	ok, err = p.HasOrgPermissionOn(ctx, "u1", "o1", PermOrgManage, "campus", "c1")
	if err != nil || !ok {
		t.Fatalf("HasOrgPermissionOn(org:manage @ campus/c1) = %v, %v; want true", ok, err)
	}
	// ...and appears in the effective permission set.
	perms, err := p.GetMemberPermissions(ctx, "o1", "u1")
	if err != nil {
		t.Fatalf("GetMemberPermissions: %v", err)
	}
	found := false
	for _, perm := range perms.Permissions {
		if perm == PermOrgManage {
			found = true
		}
	}
	if !found || perms.Role != orgtypes.RoleMember {
		t.Fatalf("effective perms = %+v (role %q), want org:manage included and primary role untouched", perms.Permissions, perms.Role)
	}

	// The listing marks the membership entry as primary.
	list, err := p.ListRoleAssignments(ctx, "o1", "u1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 3 || !list[0].Primary || list[0].Scope != nil || list[1].Scope != nil || list[1].Primary || list[2].Scope == nil {
		t.Fatalf("list = %+v, want [primary, unscoped extra, scoped extra]", list)
	}

	// The reserved owner role cannot be granted through either API.
	if err := p.AddRoleAssignment(ctx, "o1", "u1", orgtypes.RoleOwner, nil); !errors.Is(err, ErrOwnerNotAssignable) {
		t.Fatalf("add owner: got %v, want ErrOwnerNotAssignable", err)
	}
	if err := p.AssignRole(ctx, "o1", "u1", orgtypes.RoleOwner, &orgtypes.Scope{Type: "campus", ID: "c1"}); !errors.Is(err, ErrOwnerNotAssignable) {
		t.Fatalf("assign owner scoped: got %v, want ErrOwnerNotAssignable", err)
	}

	// Non-members cannot receive assignments.
	if err := New(nil, &fakeStore{}).AddRoleAssignment(ctx, "o1", "u1", "member", nil); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("non-member add: got %v, want sql.ErrNoRows", err)
	}
}

func TestPermissionCatalog(t *testing.T) {
	p := New(&Config{Permissions: map[Permission]PermissionDefinition{
		"billing:view":    {Description: "View invoices", Module: "billing"},
		"billing:approve": {Description: "Approve invoices", Module: "billing", Sensitive: true},
	}}, &fakeStore{})

	byPerm := map[Permission]PermissionInfo{}
	for _, info := range p.PermissionCatalog() {
		byPerm[info.Permission] = info
	}
	if len(byPerm) != len(frameworkPermissions)+2 {
		t.Fatalf("catalog size = %d, want %d", len(byPerm), len(frameworkPermissions)+2)
	}
	if info := byPerm[PermOrgDelete]; !info.Framework || !info.Sensitive {
		t.Errorf("framework permission = %+v, want Framework and Sensitive", info)
	}
	if info := byPerm["billing:view"]; info.Framework || info.Sensitive || info.Module != "billing" || info.Description != "View invoices" {
		t.Errorf("app permission = %+v", info)
	}
	if info := byPerm["billing:approve"]; !info.Sensitive {
		t.Errorf("sensitive app permission not flagged: %+v", info)
	}

	// Registering a framework permission supplies metadata without changing
	// its flags.
	p2 := New(&Config{Permissions: map[Permission]PermissionDefinition{
		PermOrgView: {Description: "See the org", Module: "core"},
	}}, &fakeStore{})
	for _, info := range p2.PermissionCatalog() {
		if info.Permission != PermOrgView {
			continue
		}
		if !info.Framework || !info.Sensitive || info.Module != "core" || info.Description != "See the org" {
			t.Errorf("registered framework entry = %+v", info)
		}
	}
}

func TestPermissionCatalogValidation(t *testing.T) {
	ctx := context.Background()

	// Without a catalog every permission string is accepted (unchanged).
	open := New(nil, &fakeStore{memberOrgRole: orgtypes.RoleOwner})
	if _, err := open.CreateRole(ctx, "o1", "custom", []Permission{"anything:goes"}); err != nil {
		t.Fatalf("no catalog: %v", err)
	}

	cfg := &Config{Permissions: map[Permission]PermissionDefinition{
		"billing:view":    {Module: "billing"},
		"billing:approve": {Module: "billing", Sensitive: true},
	}}
	p := New(cfg, &fakeStore{memberOrgRole: orgtypes.RoleOwner})

	if _, err := p.CreateRole(ctx, "o1", "custom", []Permission{"billing:view", "ghost"}); !errors.Is(err, ErrUnknownPermission) {
		t.Fatalf("create role with unknown permission: got %v, want ErrUnknownPermission", err)
	}
	if _, err := p.UpdateRole(ctx, "o1", "billing", []Permission{"ghost"}); !errors.Is(err, ErrUnknownPermission) {
		t.Fatalf("update role with unknown permission: got %v, want ErrUnknownPermission", err)
	}
	if err := p.SetMemberPermissionOverrides(ctx, "o1", "u1", []orgtypes.MemberPermissionOverride{
		{Permission: "ghost", Effect: orgtypes.PermissionEffectGrant},
	}); !errors.Is(err, ErrUnknownPermission) {
		t.Fatalf("override with unknown permission: got %v, want ErrUnknownPermission", err)
	}
	// Framework and cataloged permissions are accepted.
	if _, err := p.CreateRole(ctx, "o1", "billing", []Permission{PermOrgView, "billing:approve"}); err != nil {
		t.Fatalf("known permissions: %v", err)
	}
}

func TestSensitivePermissionIsCapped(t *testing.T) {
	ctx := context.Background()
	cfg := &Config{
		Permissions: map[Permission]PermissionDefinition{
			"billing:approve": {Sensitive: true},
			"billing:view":    {},
			PermOrgDelete:     {}, // cannot loosen a framework permission
		},
		OrgRoles: map[string]RoleDefinition{
			"billing-holder": {Permissions: []Permission{"billing:approve"}},
		},
	}
	admin := New(cfg, &fakeStore{memberOrgRole: orgtypes.RoleAdmin})

	// A sensitive app permission is capped like a framework one.
	if bad, err := admin.ungrantablePermission(ctx, "u1", "o1", []Permission{"billing:approve"}); err != nil || bad != "billing:approve" {
		t.Fatalf("sensitive permission: bad=%q err=%v, want capped", bad, err)
	}
	// Non-sensitive app permissions stay uncapped.
	if bad, err := admin.ungrantablePermission(ctx, "u1", "o1", []Permission{"billing:view"}); err != nil || bad != "" {
		t.Fatalf("plain app permission: bad=%q err=%v, want grantable", bad, err)
	}
	// Registering a framework permission never loosens its cap.
	if bad, err := admin.ungrantablePermission(ctx, "u1", "o1", []Permission{PermOrgDelete}); err != nil || bad != PermOrgDelete {
		t.Fatalf("framework permission: bad=%q err=%v, want capped", bad, err)
	}
	// A member holding the sensitive permission may grant it.
	holder := New(cfg, &fakeStore{memberOrgRole: "billing-holder"})
	if bad, err := holder.ungrantablePermission(ctx, "u1", "o1", []Permission{"billing:approve"}); err != nil || bad != "" {
		t.Fatalf("holder granting sensitive permission: bad=%q err=%v, want grantable", bad, err)
	}
}

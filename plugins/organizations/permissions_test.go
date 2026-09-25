package organizations

import (
	"context"
	"database/sql"
	"errors"
	"testing"

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
}

func (f *fakeStore) GetMember(_ context.Context, _, _ string) (orgtypes.Member, error) {
	if f.memberOrgRole == "" {
		return orgtypes.Member{}, sql.ErrNoRows
	}
	return orgtypes.Member{Role: f.memberOrgRole}, nil
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

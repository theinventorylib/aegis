package organizations

import (
	"context"
	"database/sql"
	"testing"

	orgtypes "github.com/theinventorylib/aegis/v2/plugins/organizations/types"
)

// fakeStore embeds the interface so only the methods used by a test need to be
// implemented.
type fakeStore struct {
	orgtypes.OrganizationStore
	memberOrgRole string
	teamRole      string
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

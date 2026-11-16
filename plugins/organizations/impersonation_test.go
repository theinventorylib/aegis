package organizations

import (
	"context"
	"errors"
	"testing"

	"github.com/theinventorylib/aegis/v2/auth"
	"github.com/theinventorylib/aegis/v2/core"
	orgtypes "github.com/theinventorylib/aegis/v2/plugins/organizations/types"
)

func impersonatedContext(orgID string) context.Context {
	session := &auth.Session{
		ID: "s-imp",
		Impersonation: &auth.Impersonation{
			SessionID:      "s-imp",
			ActorUserID:    "platform-admin",
			OrganizationID: orgID,
		},
	}
	return core.WithSession(context.Background(), session)
}

func impersonationTestPlugin() (*Plugin, *keyedStore) {
	store := &keyedStore{roles: map[string]string{
		"owner":  orgtypes.RoleOwner,
		"admin":  orgtypes.RoleAdmin,
		"target": "billing-admin",
	}}
	p := New(&Config{
		OrgRoles: map[string]RoleDefinition{
			"billing-admin": {Permissions: []Permission{PermOrgView, PermMemberView, "billing:manage"}},
		},
		Permissions: map[Permission]PermissionDefinition{
			"billing:manage": {Module: "billing", Sensitive: true, Description: "Manage billing"},
		},
	}, store)
	return p, store
}

func TestImpersonationDeniesSensitivePermissions(t *testing.T) {
	p, _ := impersonationTestPlugin()
	ctx := impersonatedContext("o1")

	// The target's normal role grants billing; impersonation must still deny it.
	if ok, err := p.HasOrgPermission(context.Background(), "target", "o1", "billing:manage"); err != nil || !ok {
		t.Fatalf("precondition: target should hold billing (ok=%v err=%v)", ok, err)
	}
	if ok, err := p.HasOrgPermission(ctx, "target", "o1", "billing:manage"); err != nil || ok {
		t.Fatalf("impersonation must deny sensitive permission (ok=%v err=%v)", ok, err)
	}
	if ok, err := p.HasOrgPermission(ctx, "target", "o1", PermOrgView); err != nil || !ok {
		t.Fatalf("impersonation should keep ordinary read permissions (ok=%v err=%v)", ok, err)
	}
	if ok, _ := p.HasOrgPermission(ctx, "target", "other-org", PermOrgView); ok {
		t.Fatal("impersonation must be scoped to its organization")
	}
	if ok, _ := p.HasOrgSensitivePermission(ctx, "target", "o1", "billing:manage"); ok {
		t.Fatal("HasOrgSensitivePermission must deny impersonation")
	}
	if ok, err := p.HasOrgSensitivePermission(context.Background(), "target", "o1", "billing:manage"); err != nil || !ok {
		t.Fatalf("non-impersonated sensitive check should pass (ok=%v err=%v)", ok, err)
	}
	if ok, _ := p.HasOrgSensitivePermission(context.Background(), "target", "o1", PermOrgView); ok {
		t.Fatal("HasOrgSensitivePermission must reject non-sensitive permissions")
	}

	ownerPlugin := New(nil, &keyedStore{roles: map[string]string{"target": orgtypes.RoleOwner}})
	if ok, _ := ownerPlugin.HasOrgPermission(ctx, "target", "o1", PermOrgDelete); ok {
		t.Fatal("organization deletion must be denied during impersonation")
	}
}

func TestImpersonationFiltersMemberPermissions(t *testing.T) {
	p, _ := impersonationTestPlugin()

	normal, err := p.GetMemberPermissions(context.Background(), "target", "o1")
	if err != nil {
		t.Fatalf("normal permissions: %v", err)
	}
	if !containsPermission(normal.Permissions, "billing:manage") {
		t.Fatal("precondition: normal permissions should include billing")
	}

	filtered, err := p.GetMemberPermissions(impersonatedContext("o1"), "target", "o1")
	if err != nil {
		t.Fatalf("impersonated permissions: %v", err)
	}
	if containsPermission(filtered.Permissions, "billing:manage") {
		t.Fatal("impersonated permission list must hide sensitive permissions")
	}
	if !containsPermission(filtered.Permissions, PermOrgView) {
		t.Fatal("ordinary permissions should remain visible")
	}
}

func TestSensitivePermissionEditsAreOwnerOnlyAndNeverImpersonated(t *testing.T) {
	p, _ := impersonationTestPlugin()
	overrides := []orgtypes.MemberPermissionOverride{{
		Permission: "billing:manage",
		Effect:     orgtypes.PermissionEffectGrant,
	}}

	if err := p.SetSensitiveMemberPermissionOverrides(context.Background(), "admin", "o1", "target", overrides); !errors.Is(err, ErrSensitiveOwnerRequired) {
		t.Fatalf("admin edit: got %v, want ErrSensitiveOwnerRequired", err)
	}
	if err := p.SetSensitiveMemberPermissionOverrides(impersonatedContext("o1"), "owner", "o1", "target", overrides); !errors.Is(err, ErrImpersonationForbidden) {
		t.Fatalf("impersonated edit: got %v, want ErrImpersonationForbidden", err)
	}
	if err := p.SetSensitiveMemberPermissionOverrides(context.Background(), "owner", "o1", "target", []orgtypes.MemberPermissionOverride{{
		Permission: string(PermOrgView),
		Effect:     orgtypes.PermissionEffectGrant,
	}}); !errors.Is(err, ErrNotSensitivePermission) {
		t.Fatalf("non-sensitive edit: got %v, want ErrNotSensitivePermission", err)
	}
	if err := p.SetSensitiveMemberPermissionOverrides(context.Background(), "owner", "o1", "target", overrides); err != nil {
		t.Fatalf("owner should be able to edit sensitive permissions: %v", err)
	}
}

func TestSensitivePermissionsList(t *testing.T) {
	p, _ := impersonationTestPlugin()
	got := p.SensitivePermissions()
	if !containsPermission(got, "billing:manage") {
		t.Fatalf("SensitivePermissions() = %v, want billing:manage", got)
	}
	if !p.IsSensitivePermission("billing:manage") {
		t.Fatal("IsSensitivePermission should report the catalog flag")
	}
	if p.IsSensitivePermission(PermOrgView) {
		t.Fatal("framework permissions are not app-sensitive")
	}
}

func containsPermission(perms []Permission, want Permission) bool {
	for _, perm := range perms {
		if perm == want {
			return true
		}
	}
	return false
}

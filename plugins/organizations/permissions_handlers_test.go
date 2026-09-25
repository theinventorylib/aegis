package organizations

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/theinventorylib/aegis/v2/auth"
	"github.com/theinventorylib/aegis/v2/core"
	orgtypes "github.com/theinventorylib/aegis/v2/plugins/organizations/types"
)

// keyedStore resolves a member's role per user so a test can model an actor and
// a differently-roled target in the same organization.
type keyedStore struct {
	orgtypes.OrganizationStore
	roles     map[string]string
	overrides []orgtypes.MemberPermissionOverride
}

func (s *keyedStore) GetMember(_ context.Context, userID, _ string) (orgtypes.Member, error) {
	role, ok := s.roles[userID]
	if !ok {
		return orgtypes.Member{}, sql.ErrNoRows
	}
	return orgtypes.Member{Role: role}, nil
}

func (s *keyedStore) HasOrgRole(_ context.Context, userID, _ string, roles ...string) (bool, error) {
	role, ok := s.roles[userID]
	if !ok {
		return false, nil
	}
	for _, r := range roles {
		if r == role {
			return true, nil
		}
	}
	return false, nil
}

func (s *keyedStore) ListMemberPermissionOverrides(_ context.Context, _, _ string) ([]orgtypes.MemberPermissionOverride, error) {
	return s.overrides, nil
}

func (s *keyedStore) DeleteMemberPermissionOverrides(_ context.Context, _, _ string) error {
	s.overrides = nil
	return nil
}

func (s *keyedStore) CreateMemberPermissionOverride(_ context.Context, o orgtypes.MemberPermissionOverride) error {
	s.overrides = append(s.overrides, o)
	return nil
}

func putMemberPermissions(t *testing.T, p *Plugin, targetID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/organizations/o1/members/"+targetID+"/permissions", strings.NewReader(body))
	req.SetPathValue("id", "o1")
	req.SetPathValue("userId", targetID)
	req = req.WithContext(core.WithUser(req.Context(), &auth.User{ID: "admin"}))
	rec := httptest.NewRecorder()
	p.UpdateMemberPermissionsHandler(rec, req)
	return rec
}

func TestUpdateMemberPermissionsCapsGrants(t *testing.T) {
	p := New(nil, &keyedStore{roles: map[string]string{"admin": orgtypes.RoleAdmin, "member": orgtypes.RoleMember}})

	// An admin may not grant org:delete: the built-in admin role lacks it.
	rec := putMemberPermissions(t, p, "member", `{"overrides":[{"permission":"org:delete","effect":"grant"}]}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("granting org:delete: status = %d, want 403 (%s)", rec.Code, rec.Body.String())
	}

	// Normalization must not slip a framework permission past the cap: each of
	// these sanitizes to "org:delete" on the write path.
	for _, raw := range []string{" org:delete", "org:delete ", "org:\\u0000delete"} {
		rec = putMemberPermissions(t, p, "member", `{"overrides":[{"permission":"`+raw+`","effect":"grant"}]}`)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("padded %q: status = %d, want 403 (%s)", raw, rec.Code, rec.Body.String())
		}
	}

	// A permission the actor holds is grantable.
	rec = putMemberPermissions(t, p, "member", `{"overrides":[{"permission":"member:manage","effect":"grant"}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("granting member:manage: status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
}

func TestUpdateMemberPermissionsProtectsOwner(t *testing.T) {
	p := New(nil, &keyedStore{roles: map[string]string{"admin": orgtypes.RoleAdmin, "owner": orgtypes.RoleOwner}})

	rec := putMemberPermissions(t, p, "owner", `{"overrides":[{"permission":"org:delete","effect":"deny"}]}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("overriding owner: status = %d, want 400 (%s)", rec.Code, rec.Body.String())
	}
}

func TestUngrantablePermission(t *testing.T) {
	ctx := context.Background()
	p := New(nil, &fakeStore{memberOrgRole: orgtypes.RoleAdmin})

	if bad, err := p.ungrantablePermission(ctx, "u1", "o1", []Permission{PermOrgDelete}); err != nil || bad != PermOrgDelete {
		t.Fatalf("framework permission: bad=%q err=%v, want %q", bad, err, PermOrgDelete)
	}
	// A normalized spelling of a framework permission is capped identically.
	if bad, err := p.ungrantablePermission(ctx, "u1", "o1", []Permission{" org:delete "}); err != nil || bad != PermOrgDelete {
		t.Fatalf("padded framework permission: bad=%q err=%v, want %q", bad, err, PermOrgDelete)
	}
	if bad, err := p.ungrantablePermission(ctx, "u1", "o1", []Permission{PermMemberManage}); err != nil || bad != "" {
		t.Fatalf("held permission: bad=%q err=%v, want grantable", bad, err)
	}
	// App-defined permissions are not capped by the framework.
	if bad, err := p.ungrantablePermission(ctx, "u1", "o1", []Permission{"billing:manage"}); err != nil || bad != "" {
		t.Fatalf("app permission: bad=%q err=%v, want grantable", bad, err)
	}
}

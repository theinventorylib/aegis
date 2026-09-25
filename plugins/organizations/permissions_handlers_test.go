package organizations

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	"github.com/labstack/echo/v4"

	"github.com/theinventorylib/aegis/v2/auth"
	"github.com/theinventorylib/aegis/v2/core"
	orgtypes "github.com/theinventorylib/aegis/v2/plugins/organizations/types"
	aegisrouter "github.com/theinventorylib/aegis/v2/router"
	"github.com/theinventorylib/aegis/v2/router/routers"
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

func (s *keyedStore) ListOrganizationMembers(_ context.Context, _ string, _, _ int) ([]orgtypes.Member, error) {
	members := make([]orgtypes.Member, 0, len(s.roles))
	for userID, role := range s.roles {
		members = append(members, orgtypes.Member{UserID: userID, Role: role})
	}
	return members, nil
}

func (s *keyedStore) CountOrganizationMembers(_ context.Context, _ string) (int, error) {
	return len(s.roles), nil
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

func getMyPermissions(t *testing.T, p *Plugin, userID, orgID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/organizations/"+orgID+"/permissions", nil)
	req.SetPathValue("id", orgID)
	req = req.WithContext(core.WithUser(req.Context(), &auth.User{ID: userID}))
	rec := httptest.NewRecorder()
	p.GetMyPermissionsHandler(rec, req)
	return rec
}

func TestGetMyPermissionsHandler(t *testing.T) {
	p := New(nil, &keyedStore{roles: map[string]string{"member": orgtypes.RoleMember}})

	rec := getMyPermissions(t, p, "member", "o1")
	if rec.Code != http.StatusOK {
		t.Fatalf("member: status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"role":"member"`) {
		t.Fatalf("body missing role: %s", rec.Body.String())
	}

	// A non-member cannot read permissions for an organization they are not in.
	rec = getMyPermissions(t, p, "outsider", "o1")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-member: status = %d, want 403 (%s)", rec.Code, rec.Body.String())
	}
}

func listMembersPermissions(t *testing.T, p *Plugin, actorID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/organizations/o1/members/permissions", nil)
	req.SetPathValue("id", "o1")
	req = req.WithContext(core.WithUser(req.Context(), &auth.User{ID: actorID}))
	rec := httptest.NewRecorder()
	p.ListMembersPermissionsHandler(rec, req)
	return rec
}

func TestListMembersPermissionsHandler(t *testing.T) {
	p := New(nil, &keyedStore{roles: map[string]string{
		"admin": orgtypes.RoleAdmin, "member": orgtypes.RoleMember, "owner": orgtypes.RoleOwner,
	}})

	rec := listMembersPermissions(t, p, "admin")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin: status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{`"userId":"admin"`, `"userId":"member"`, `"userId":"owner"`, `"totalCount":3`} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %s: %s", want, body)
		}
	}

	// A plain member lacks member:assign_roles and cannot enumerate the org.
	rec = listMembersPermissions(t, p, "member")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member: status = %d, want 403 (%s)", rec.Code, rec.Body.String())
	}
}

// The new permission routes must coexist with the :userId variants on every
// supported router (static segment wins over the parameter).
func TestOrganizationPermissionRoutesRegisterOnAllRouters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mountAndProbe := func(t *testing.T, r aegisrouter.Router) {
		t.Helper()
		p := New(nil, nil)
		p.MountRoutes(r, "/auth/organizations")
		for _, path := range []string{
			"/auth/organizations/o1/permissions",
			"/auth/organizations/o1/members/permissions",
		} {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			// Registered behind RequireAuthMiddleware, so an anonymous request
			// is 401; a 404 would mean the route never matched.
			if rec.Code == http.StatusNotFound {
				t.Fatalf("%s: route not matched (404)", path)
			}
		}
	}

	t.Run("chi", func(t *testing.T) { mountAndProbe(t, routers.NewChiRouter(chi.NewRouter())) })
	t.Run("echo", func(t *testing.T) { mountAndProbe(t, routers.NewEchoRouter(echo.New())) })
	t.Run("gin", func(t *testing.T) { mountAndProbe(t, routers.NewGinRouter(gin.New())) })
}

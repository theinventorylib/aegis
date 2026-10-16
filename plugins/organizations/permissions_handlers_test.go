package organizations

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	teamRoles map[string]string
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

func (s *keyedStore) ListOrganizationRoles(_ context.Context, _ string) ([]orgtypes.OrganizationRole, error) {
	return nil, nil
}

func (s *keyedStore) GetTeam(_ context.Context, id string) (orgtypes.Team, error) {
	return orgtypes.Team{ID: id, OrganizationID: "o1"}, nil
}

func (s *keyedStore) ListMemberRoleAssignments(_ context.Context, _, _ string) ([]orgtypes.RoleAssignment, error) {
	return nil, nil
}

func (s *keyedStore) CreateMember(_ context.Context, _, userID, _, role string, _, _ time.Time) error {
	s.roles[userID] = role
	return nil
}

func (s *keyedStore) UpdateMemberRole(_ context.Context, userID, _, role string, _ time.Time) error {
	s.roles[userID] = role
	return nil
}

func (s *keyedStore) CreateInvitation(_ context.Context, _ orgtypes.Invitation) error {
	return nil
}

func (s *keyedStore) GetTeamMember(_ context.Context, teamID, userID string) (orgtypes.TeamMember, error) {
	role, ok := s.teamRoles[userID]
	if !ok {
		return orgtypes.TeamMember{}, sql.ErrNoRows
	}
	return orgtypes.TeamMember{TeamID: teamID, UserID: userID, Role: role}, nil
}

func (s *keyedStore) CreateTeamMember(_ context.Context, _, userID, _, role string, _, _ time.Time) error {
	s.teamRoles[userID] = role
	return nil
}

func (s *keyedStore) UpdateTeamMemberRole(_ context.Context, _, userID, role string, _ time.Time) error {
	s.teamRoles[userID] = role
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

// TestAssignRoleCapsActorAuthority pins the assignment-time cap: the same
// authority rule that guards permission edits must guard role assignment on
// member add, member role change and invitation creation.
func TestAssignRoleCapsActorAuthority(t *testing.T) {
	store := &keyedStore{roles: map[string]string{
		"admin":   orgtypes.RoleAdmin,
		"member":  orgtypes.RoleMember,
		"inviter": "inviter",
		"target":  orgtypes.RoleMember,
	}}
	p := New(&Config{OrgRoles: map[string]RoleDefinition{
		// Carries org:delete, which the built-in admin role lacks.
		"superadmin": {Permissions: []Permission{PermOrgView, PermOrgDelete}},
		// Can invite, but holds only read permissions beyond that.
		"inviter": {Permissions: []Permission{PermInvitationManage, PermOrgView, PermMemberView, PermTeamView}},
	}}, store)

	addMember := func(actor, role string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/organizations/o1/members",
			strings.NewReader(`{"userId":"target","role":"`+role+`"}`))
		req.SetPathValue("id", "o1")
		req = req.WithContext(core.WithUser(req.Context(), &auth.User{ID: actor}))
		rec := httptest.NewRecorder()
		p.AddOrganizationMemberHandler(rec, req)
		return rec
	}
	changeRole := func(actor, role string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, "/organizations/o1/members/target",
			strings.NewReader(`{"role":"`+role+`"}`))
		req.SetPathValue("id", "o1")
		req.SetPathValue("userId", "target")
		req = req.WithContext(core.WithUser(req.Context(), &auth.User{ID: actor}))
		rec := httptest.NewRecorder()
		p.UpdateMemberRoleHandler(rec, req)
		return rec
	}
	invite := func(actor, path, body string, pathParams map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		for k, v := range pathParams {
			req.SetPathValue(k, v)
		}
		req = req.WithContext(core.WithUser(req.Context(), &auth.User{ID: actor}))
		rec := httptest.NewRecorder()
		p.CreateInvitationHandler(rec, req)
		return rec
	}

	// Member add: an admin cannot grant a role carrying org:delete...
	rec := addMember("admin", "superadmin")
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "org:delete") {
		t.Fatalf("add superadmin: status = %d body = %s, want 403 naming org:delete", rec.Code, rec.Body.String())
	}
	// ...but may grant a role within its own authority.
	if rec := addMember("admin", "member"); rec.Code != http.StatusCreated {
		t.Fatalf("add member: status = %d, want 201 (%s)", rec.Code, rec.Body.String())
	}
	if store.roles["target"] != "member" {
		t.Fatalf("target role = %q, want member", store.roles["target"])
	}

	// Role change: same cap.
	rec = changeRole("admin", "superadmin")
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "org:delete") {
		t.Fatalf("update to superadmin: status = %d body = %s, want 403 naming org:delete", rec.Code, rec.Body.String())
	}
	if rec := changeRole("admin", "admin"); rec.Code != http.StatusOK {
		t.Fatalf("update to admin: status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}

	// Invitations: the invited role is capped as a grant.
	rec = invite("inviter", "/organizations/o1/invitations", `{"email":"invitee@example.com","role":"superadmin"}`, map[string]string{"id": "o1"})
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "org:delete") {
		t.Fatalf("invite superadmin: status = %d body = %s, want 403 naming org:delete", rec.Code, rec.Body.String())
	}
	if rec := invite("inviter", "/organizations/o1/invitations", `{"email":"invitee@example.com","role":"member"}`, map[string]string{"id": "o1"}); rec.Code != http.StatusCreated {
		t.Fatalf("invite member: status = %d, want 201 (%s)", rec.Code, rec.Body.String())
	}
	// Team invitations use the team role set: the inviter may grant a
	// member-viewing team role but not a lead role they cannot grant themselves.
	rec = invite("inviter", "/teams/t1/invitations", `{"email":"invitee@example.com","teamId":"t1","role":"lead"}`, map[string]string{"teamId": "t1"})
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "team_member:manage") {
		t.Fatalf("invite team lead: status = %d body = %s, want 403 naming team_member:manage", rec.Code, rec.Body.String())
	}
	if rec := invite("inviter", "/teams/t1/invitations", `{"email":"invitee@example.com","teamId":"t1","role":"member"}`, map[string]string{"teamId": "t1"}); rec.Code != http.StatusCreated {
		t.Fatalf("invite team member: status = %d, want 201 (%s)", rec.Code, rec.Body.String())
	}
}

func TestRequireOrgPermissionOn(t *testing.T) {
	p := New(&Config{OrgRoles: map[string]RoleDefinition{
		"regional-lead": {Permissions: []Permission{PermOrgView, PermMemberManage}},
	}}, &fakeStore{
		memberOrgRole: orgtypes.RoleMember,
		assignments: []orgtypes.RoleAssignment{{
			ID: "a1", Role: "regional-lead", Scope: &orgtypes.Scope{Type: "region", ID: "r1"},
		}},
	})

	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	h := p.RequireOrgPermissionOn(PermMemberManage, func(r *http.Request) (string, string) {
		return "region", r.PathValue("regionID")
	})(next)

	do := func(regionID string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/organizations/o1/regions/"+regionID+"/reports", nil)
		req.SetPathValue("id", "o1")
		req.SetPathValue("regionID", regionID)
		req = req.WithContext(core.WithUser(req.Context(), &auth.User{ID: "u1"}))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	// The scoped assignment grants the permission on its own scope only.
	if rec := do("r1"); rec.Code != http.StatusOK || !called {
		t.Fatalf("scoped holder: status = %d called = %v, want 200 and next", rec.Code, called)
	}
	if rec := do("r2"); rec.Code != http.StatusForbidden {
		t.Fatalf("other scope: status = %d, want 403 (%s)", rec.Code, rec.Body.String())
	}
	// An unresolvable scope is a bad request, not a permission failure.
	called = false
	if rec := do(""); rec.Code != http.StatusBadRequest || called {
		t.Fatalf("missing scope: status = %d called = %v, want 400", rec.Code, called)
	}
}

// TestTeamMemberRoleCapsActorAuthority pins the cap on the team membership
// paths: a team lead keeps managing team roles within their own authority but
// cannot grant a team role carrying a permission they do not hold.
func TestTeamMemberRoleCapsActorAuthority(t *testing.T) {
	store := &keyedStore{
		roles:     map[string]string{"admin": orgtypes.RoleAdmin, "lead-user": orgtypes.RoleMember, "target": orgtypes.RoleMember},
		teamRoles: map[string]string{"lead-user": orgtypes.RoleTeamLead},
	}
	p := New(&Config{TeamRoles: map[string]RoleDefinition{
		"co-lead": {Permissions: []Permission{PermTeamManage}},
	}}, store)

	add := func(actor, role string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/teams/t1/members",
			strings.NewReader(`{"userId":"target","role":"`+role+`"}`))
		req.SetPathValue("teamId", "t1")
		req = req.WithContext(core.WithUser(req.Context(), &auth.User{ID: actor}))
		rec := httptest.NewRecorder()
		p.AddTeamMemberHandler(rec, req)
		return rec
	}
	update := func(actor, role string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, "/teams/t1/members/target",
			strings.NewReader(`{"role":"`+role+`"}`))
		req.SetPathValue("teamId", "t1")
		req.SetPathValue("userId", "target")
		req = req.WithContext(core.WithUser(req.Context(), &auth.User{ID: actor}))
		rec := httptest.NewRecorder()
		p.UpdateTeamMemberRoleHandler(rec, req)
		return rec
	}

	// A team lead may grant roles within their own (org + team) authority.
	if rec := add("lead-user", orgtypes.RoleTeamLead); rec.Code != http.StatusCreated {
		t.Fatalf("lead grants lead: status = %d, want 201 (%s)", rec.Code, rec.Body.String())
	}
	// ...but not a team role carrying a permission they do not hold.
	rec := add("lead-user", "co-lead")
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "team:manage") {
		t.Fatalf("lead grants co-lead: status = %d body = %s, want 403 naming team:manage", rec.Code, rec.Body.String())
	}
	rec = update("lead-user", "co-lead")
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "team:manage") {
		t.Fatalf("lead updates to co-lead: status = %d body = %s, want 403 naming team:manage", rec.Code, rec.Body.String())
	}
	// Org admins hold the team permissions and may grant them.
	if rec := update("admin", "co-lead"); rec.Code != http.StatusOK {
		t.Fatalf("admin updates to co-lead: status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
}

// TestCreateRoleRejectsUnknownPermissionFromCatalog pins the 400 mapping for
// catalog validation failures.
func TestCreateRoleRejectsUnknownPermissionFromCatalog(t *testing.T) {
	p := New(&Config{Permissions: map[Permission]PermissionDefinition{
		"billing:view": {},
	}}, &keyedStore{roles: map[string]string{"admin": orgtypes.RoleAdmin}})

	req := httptest.NewRequest(http.MethodPost, "/organizations/o1/roles",
		strings.NewReader(`{"name":"billing","permissions":["ghost"]}`))
	req.SetPathValue("id", "o1")
	req = req.WithContext(core.WithUser(req.Context(), &auth.User{ID: "admin"}))
	rec := httptest.NewRecorder()
	p.CreateRoleHandler(rec, req)

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "unknown permission") {
		t.Fatalf("status = %d body = %s, want 400 naming the unknown permission", rec.Code, rec.Body.String())
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

// The permission, role-assignment and rename routes must coexist with the
// :userId/:name variants on every supported router (static segment wins over
// the parameter).
func TestOrganizationPermissionRoutesRegisterOnAllRouters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mountAndProbe := func(t *testing.T, r aegisrouter.Router) {
		t.Helper()
		p := New(nil, nil)
		p.MountRoutes(r, "/auth/organizations")
		for _, route := range []struct{ method, path string }{
			{http.MethodGet, "/auth/organizations/o1/permissions"},
			{http.MethodGet, "/auth/organizations/o1/members/permissions"},
			{http.MethodGet, "/auth/organizations/o1/members/u1/role-assignments"},
			{http.MethodPost, "/auth/organizations/o1/members/u1/role-assignments"},
			{http.MethodDelete, "/auth/organizations/o1/members/u1/role-assignments/a1"},
			{http.MethodPost, "/auth/organizations/o1/roles/lorem/rename"},
		} {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(route.method, route.path, nil))
			// Registered behind RequireAuthMiddleware, so an anonymous request
			// is 401; a 404 would mean the route never matched.
			if rec.Code == http.StatusNotFound {
				t.Fatalf("%s %s: route not matched (404)", route.method, route.path)
			}
		}
	}

	t.Run("chi", func(t *testing.T) { mountAndProbe(t, routers.NewChiRouter(chi.NewRouter())) })
	t.Run("echo", func(t *testing.T) { mountAndProbe(t, routers.NewEchoRouter(echo.New())) })
	t.Run("gin", func(t *testing.T) { mountAndProbe(t, routers.NewGinRouter(gin.New())) })
}

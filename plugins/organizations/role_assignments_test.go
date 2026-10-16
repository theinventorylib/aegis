package organizations

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/theinventorylib/aegis/v2/auth"
	"github.com/theinventorylib/aegis/v2/core"
	orgtypes "github.com/theinventorylib/aegis/v2/plugins/organizations/types"
)

// assignmentStore is a fake OrganizationStore for the role-assignment and
// rename surfaces: members with primary roles, custom roles, and assignment
// rows. Unimplemented methods panic through the embedded interface.
type assignmentStore struct {
	orgtypes.OrganizationStore
	primary     map[string]string // userID → primary role
	custom      map[string]orgtypes.OrganizationRole
	assignments map[string][]orgtypes.RoleAssignment // userID → additional rows
	invitations []orgtypes.Invitation
	createErr   error // when set, CreateMemberRoleAssignment fails
}

func newAssignmentStore() *assignmentStore {
	return &assignmentStore{
		primary:     map[string]string{},
		custom:      map[string]orgtypes.OrganizationRole{},
		assignments: map[string][]orgtypes.RoleAssignment{},
	}
}

func (s *assignmentStore) GetMember(_ context.Context, userID, orgID string) (orgtypes.Member, error) {
	role, ok := s.primary[userID]
	if !ok {
		return orgtypes.Member{}, sql.ErrNoRows
	}
	return orgtypes.Member{ID: "m-" + userID, UserID: userID, OrganizationID: orgID, Role: role}, nil
}

func (s *assignmentStore) ListMemberPermissionOverrides(_ context.Context, _, _ string) ([]orgtypes.MemberPermissionOverride, error) {
	return nil, nil
}

func (s *assignmentStore) HasOrgRole(_ context.Context, userID, _ string, roles ...string) (bool, error) {
	role, ok := s.primary[userID]
	if !ok {
		return false, nil
	}
	for _, want := range roles {
		if want == role {
			return true, nil
		}
	}
	return false, nil
}

func (s *assignmentStore) ListMemberRoleAssignments(_ context.Context, _, userID string) ([]orgtypes.RoleAssignment, error) {
	return s.assignments[userID], nil
}

func (s *assignmentStore) CreateMemberRoleAssignment(_ context.Context, id, orgID, userID, role, scopeType, scopeID string, createdAt, updatedAt time.Time) error {
	if s.createErr != nil {
		return s.createErr
	}
	var scope *orgtypes.Scope
	if scopeType != "" || scopeID != "" {
		scope = &orgtypes.Scope{Type: scopeType, ID: scopeID}
	}
	s.assignments[userID] = append(s.assignments[userID], orgtypes.RoleAssignment{
		ID: id, OrganizationID: orgID, UserID: userID, Role: role,
		Scope: scope, CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
	return nil
}

func (s *assignmentStore) DeleteMemberRoleAssignment(_ context.Context, _, userID, id string) error {
	rows := s.assignments[userID]
	for i, a := range rows {
		if a.ID == id {
			s.assignments[userID] = append(rows[:i], rows[i+1:]...)
			return nil
		}
	}
	return sql.ErrNoRows
}

func (s *assignmentStore) GetOrganizationRole(_ context.Context, _, name string) (orgtypes.OrganizationRole, error) {
	role, ok := s.custom[name]
	if !ok {
		return orgtypes.OrganizationRole{}, sql.ErrNoRows
	}
	return role, nil
}

func (s *assignmentStore) ListOrganizationRoles(_ context.Context, _ string) ([]orgtypes.OrganizationRole, error) {
	out := make([]orgtypes.OrganizationRole, 0, len(s.custom))
	for _, role := range s.custom {
		out = append(out, role)
	}
	return out, nil
}

func (s *assignmentStore) RenameOrganizationRole(_ context.Context, _, oldName, newName string, updatedAt time.Time) error {
	role, ok := s.custom[oldName]
	if !ok {
		return sql.ErrNoRows
	}
	delete(s.custom, oldName)
	role.Name = newName
	role.UpdatedAt = updatedAt
	s.custom[newName] = role
	for userID, rows := range s.assignments {
		for i := range rows {
			if rows[i].Role == oldName {
				rows[i].Role = newName
			}
		}
		s.assignments[userID] = rows
	}
	for userID, primary := range s.primary {
		if primary == oldName {
			s.primary[userID] = newName
		}
	}
	// Organization-level invitations follow the role; team-level ones carry
	// team role names and are left alone.
	for i := range s.invitations {
		if s.invitations[i].TeamID == nil && s.invitations[i].Role == oldName {
			s.invitations[i].Role = newName
		}
	}
	return nil
}

func customRole(name string, perms ...Permission) orgtypes.OrganizationRole {
	strings := make([]string, 0, len(perms))
	for _, p := range perms {
		strings = append(strings, string(p))
	}
	return orgtypes.OrganizationRole{ID: "r-" + name, Name: name, Permissions: strings}
}

func assignmentRequest(method, path, body string, params map[string]string) *http.Request {
	return assignmentRequestAs("admin", method, path, body, params)
}

func assignmentRequestAs(actorID, method, path, body string, params map[string]string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range params {
		req.SetPathValue(k, v)
	}
	return req.WithContext(core.WithUser(req.Context(), &auth.User{ID: actorID}))
}

// RenameRole carries the new name to the role row, its assignments and any
// member whose primary role it is; reserved and taken names are refused.
func TestRenameRole(t *testing.T) {
	ctx := context.Background()
	store := newAssignmentStore()
	store.custom["Analyst"] = customRole("Analyst", PermOrgView)
	store.assignments["member"] = []orgtypes.RoleAssignment{{ID: "a1", UserID: "member", Role: "Analyst"}}
	store.primary["u2"] = "Analyst"
	teamID := "t1"
	store.invitations = []orgtypes.Invitation{
		{ID: "i1", OrganizationID: "o1", Role: "Analyst"},                  // organization-level
		{ID: "i2", OrganizationID: "o1", TeamID: &teamID, Role: "Analyst"}, // team-level
	}
	p := New(nil, store)

	info, err := p.RenameRole(ctx, "o1", "Analyst", "Lead Analyst")
	if err != nil {
		t.Fatalf("RenameRole: %v", err)
	}
	if info.Name != "Lead Analyst" || !info.Custom {
		t.Fatalf("info = %+v, want the new name, custom", info)
	}
	if _, ok := store.custom["Lead Analyst"]; !ok {
		t.Fatal("role row not renamed in the store")
	}
	if got := store.assignments["member"][0].Role; got != "Lead Analyst" {
		t.Fatalf("assignment role = %q, want Lead Analyst", got)
	}
	if got := store.primary["u2"]; got != "Lead Analyst" {
		t.Fatalf("primary role = %q, want Lead Analyst", got)
	}
	if len(store.invitations) != 2 || store.invitations[0].Role != "Lead Analyst" || store.invitations[1].Role != "Analyst" {
		t.Fatalf("invitations = %+v, want the organization-level one renamed and the team-level one untouched", store.invitations)
	}

	// Renaming a role to its own name is a no-op.
	same, err := p.RenameRole(ctx, "o1", "Lead Analyst", "Lead Analyst")
	if err != nil || same.Name != "Lead Analyst" || !same.Custom {
		t.Fatalf("same-name rename: info = %+v, err = %v", same, err)
	}
	if len(store.assignments["member"]) != 1 || store.assignments["member"][0].Role != "Lead Analyst" {
		t.Fatalf("same-name rename disturbed the assignments: %+v", store.assignments["member"])
	}

	if _, err := p.RenameRole(ctx, "o1", "owner", "Boss"); !errors.Is(err, ErrRoleReserved) {
		t.Fatalf("renaming a compiled role: err = %v, want ErrRoleReserved", err)
	}
	if _, err := p.RenameRole(ctx, "o1", "Lead Analyst", "admin"); !errors.Is(err, ErrRoleReserved) {
		t.Fatalf("renaming to a compiled name: err = %v, want ErrRoleReserved", err)
	}
	store.custom["Existing"] = customRole("Existing", PermOrgView)
	if _, err := p.RenameRole(ctx, "o1", "Lead Analyst", "Existing"); !errors.Is(err, ErrRoleExists) {
		t.Fatalf("renaming onto a taken name: err = %v, want ErrRoleExists", err)
	}
	if _, err := p.RenameRole(ctx, "o1", "Missing", "Whatever"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("renaming an unknown role: err = %v, want sql.ErrNoRows", err)
	}
}

func TestAddRoleAssignmentHandler(t *testing.T) {
	store := newAssignmentStore()
	store.primary["admin"] = orgtypes.RoleAdmin
	store.primary["member"] = orgtypes.RoleMember
	store.custom["Analyst"] = customRole("Analyst", PermMemberView)
	store.custom["Dangerous"] = customRole("Dangerous", "org:delete")
	p := New(nil, store)

	// Scoped assignment on top of the member's primary role.
	rec := httptest.NewRecorder()
	p.AddRoleAssignmentHandler(rec, assignmentRequest(http.MethodPost, "/organizations/o1/members/member/role-assignments",
		`{"role":"Analyst","scopeType":"region","scopeId":"r1"}`,
		map[string]string{"id": "o1", "userId": "member"}))
	if rec.Code != http.StatusCreated {
		t.Fatalf("add assignment: status = %d, want 201 (%s)", rec.Code, rec.Body.String())
	}
	rows := store.assignments["member"]
	if len(rows) != 1 || rows[0].Role != "Analyst" || rows[0].Scope == nil || rows[0].Scope.ID != "r1" {
		t.Fatalf("assignments = %+v, want one scoped Analyst", rows)
	}
	if store.primary["member"] != orgtypes.RoleMember {
		t.Fatal("the primary role must not change")
	}

	// A permission the actor does not hold cannot be granted.
	rec = httptest.NewRecorder()
	p.AddRoleAssignmentHandler(rec, assignmentRequest(http.MethodPost, "/organizations/o1/members/member/role-assignments",
		`{"role":"Dangerous"}`, map[string]string{"id": "o1", "userId": "member"}))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("granting org:delete: status = %d, want 403 (%s)", rec.Code, rec.Body.String())
	}

	// Unknown roles are a request error, not a server error.
	rec = httptest.NewRecorder()
	p.AddRoleAssignmentHandler(rec, assignmentRequest(http.MethodPost, "/organizations/o1/members/member/role-assignments",
		`{"role":"Nope"}`, map[string]string{"id": "o1", "userId": "member"}))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown role: status = %d, want 400 (%s)", rec.Code, rec.Body.String())
	}

	// Scope halves go together.
	rec = httptest.NewRecorder()
	p.AddRoleAssignmentHandler(rec, assignmentRequest(http.MethodPost, "/organizations/o1/members/member/role-assignments",
		`{"role":"Analyst","scopeType":"region"}`, map[string]string{"id": "o1", "userId": "member"}))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("half a scope: status = %d, want 400 (%s)", rec.Code, rec.Body.String())
	}

	// The other half of the pair is symmetric: a scope id without a type.
	rec = httptest.NewRecorder()
	p.AddRoleAssignmentHandler(rec, assignmentRequest(http.MethodPost, "/organizations/o1/members/member/role-assignments",
		`{"role":"Analyst","scopeId":"r1"}`, map[string]string{"id": "o1", "userId": "member"}))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("scope id without a type: status = %d, want 400 (%s)", rec.Code, rec.Body.String())
	}

	// A whitespace-only half is blank once sanitized: it must be rejected, not
	// pass as "filled" and silently widen the assignment to organization-wide.
	rec = httptest.NewRecorder()
	p.AddRoleAssignmentHandler(rec, assignmentRequest(http.MethodPost, "/organizations/o1/members/member/role-assignments",
		`{"role":"Analyst","scopeType":"   ","scopeId":"r1"}`, map[string]string{"id": "o1", "userId": "member"}))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("whitespace scope type with a scope id: status = %d, want 400 (%s)", rec.Code, rec.Body.String())
	}
	if len(store.assignments["member"]) != 1 {
		t.Fatalf("assignments = %+v, want the rejected request not to create one", store.assignments["member"])
	}

	rec = httptest.NewRecorder()
	p.AddRoleAssignmentHandler(rec, assignmentRequest(http.MethodPost, "/organizations/o1/members/member/role-assignments",
		`{"role":"Analyst","scopeType":"region","scopeId":"   "}`, map[string]string{"id": "o1", "userId": "member"}))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("whitespace scope id with a scope type: status = %d, want 400 (%s)", rec.Code, rec.Body.String())
	}

	// A missing target member is a 404, not a role refusal.
	rec = httptest.NewRecorder()
	p.AddRoleAssignmentHandler(rec, assignmentRequest(http.MethodPost, "/organizations/o1/members/ghost/role-assignments",
		`{"role":"Analyst"}`, map[string]string{"id": "o1", "userId": "ghost"}))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("non-member target: status = %d, want 404 (%s)", rec.Code, rec.Body.String())
	}

	// A scope longer than the cross-dialect limit is a 400...
	rec = httptest.NewRecorder()
	p.AddRoleAssignmentHandler(rec, assignmentRequest(http.MethodPost, "/organizations/o1/members/member/role-assignments",
		`{"role":"Analyst","scopeType":"region","scopeId":"`+strings.Repeat("x", maxScopeLen+1)+`"}`,
		map[string]string{"id": "o1", "userId": "member"}))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("over-long scope: status = %d, want 400 (%s)", rec.Code, rec.Body.String())
	}

	// ...while a backend failure is a 500, not a client error.
	store.createErr = errors.New("boom")
	rec = httptest.NewRecorder()
	p.AddRoleAssignmentHandler(rec, assignmentRequest(http.MethodPost, "/organizations/o1/members/member/role-assignments",
		`{"role":"Analyst"}`, map[string]string{"id": "o1", "userId": "member"}))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("store failure: status = %d, want 500 (%s)", rec.Code, rec.Body.String())
	}
}

// A compiled role's name is consumer-defined and not bounded by the plugin, so
// a long one must still be assignable: the request's role is a reference to an
// existing role, not a new name.
func TestAddRoleAssignmentAcceptsLongCompiledRoleName(t *testing.T) {
	store := newAssignmentStore()
	store.primary["admin"] = orgtypes.RoleAdmin
	store.primary["member"] = orgtypes.RoleMember
	const longName = "Regional Platform Reliability Coordinator for the Northern Operations Area"
	if len(longName) <= 50 {
		t.Fatalf("the fixture must exceed the custom-role name bound (len %d)", len(longName))
	}
	p := New(&Config{OrgRoles: map[string]RoleDefinition{
		longName: {Permissions: []Permission{PermMemberView}},
	}}, store)

	rec := httptest.NewRecorder()
	p.AddRoleAssignmentHandler(rec, assignmentRequest(http.MethodPost, "/organizations/o1/members/member/role-assignments",
		`{"role":"`+longName+`"}`, map[string]string{"id": "o1", "userId": "member"}))
	if rec.Code != http.StatusCreated {
		t.Fatalf("long compiled role: status = %d, want 201 (%s)", rec.Code, rec.Body.String())
	}
	if rows := store.assignments["member"]; len(rows) != 1 || rows[0].Role != longName {
		t.Fatalf("assignments = %+v, want the long compiled role", rows)
	}
}

func TestListRoleAssignmentsHandler(t *testing.T) {
	store := newAssignmentStore()
	store.primary["admin"] = orgtypes.RoleAdmin
	store.primary["member"] = orgtypes.RoleMember
	store.assignments["member"] = []orgtypes.RoleAssignment{{ID: "a1", UserID: "member", Role: "Analyst"}}
	p := New(nil, store)

	rec := httptest.NewRecorder()
	p.ListRoleAssignmentsHandler(rec, assignmentRequest(http.MethodGet, "/organizations/o1/members/member/role-assignments",
		"", map[string]string{"id": "o1", "userId": "member"}))
	if rec.Code != http.StatusOK {
		t.Fatalf("list: status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"primary":true`) || !strings.Contains(body, "Analyst") {
		t.Fatalf("list body = %s, want the primary entry and the assignment", body)
	}

	rec = httptest.NewRecorder()
	p.ListRoleAssignmentsHandler(rec, assignmentRequest(http.MethodGet, "/organizations/o1/members/ghost/role-assignments",
		"", map[string]string{"id": "o1", "userId": "ghost"}))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("non-member: status = %d, want 404 (%s)", rec.Code, rec.Body.String())
	}

	// Authority data: member:view alone is not enough, matching the
	// member-permissions endpoint.
	store.primary["viewer"] = "Viewer"
	store.custom["Viewer"] = customRole("Viewer", PermMemberView)
	rec = httptest.NewRecorder()
	p.ListRoleAssignmentsHandler(rec, assignmentRequestAs("viewer", http.MethodGet, "/organizations/o1/members/member/role-assignments",
		"", map[string]string{"id": "o1", "userId": "member"}))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member:view only: status = %d, want 403 (%s)", rec.Code, rec.Body.String())
	}
}

func TestRemoveRoleAssignmentHandler(t *testing.T) {
	store := newAssignmentStore()
	store.primary["admin"] = orgtypes.RoleAdmin
	store.primary["member"] = orgtypes.RoleMember
	store.assignments["member"] = []orgtypes.RoleAssignment{{ID: "a1", UserID: "member", Role: "Analyst"}}
	p := New(nil, store)

	rec := httptest.NewRecorder()
	p.RemoveRoleAssignmentHandler(rec, assignmentRequest(http.MethodDelete, "/organizations/o1/members/member/role-assignments/a1",
		"", map[string]string{"id": "o1", "userId": "member", "assignmentId": "a1"}))
	if rec.Code != http.StatusOK {
		t.Fatalf("remove: status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	if len(store.assignments["member"]) != 0 {
		t.Fatalf("assignments = %+v, want none left", store.assignments["member"])
	}

	rec = httptest.NewRecorder()
	p.RemoveRoleAssignmentHandler(rec, assignmentRequest(http.MethodDelete, "/organizations/o1/members/member/role-assignments/a1",
		"", map[string]string{"id": "o1", "userId": "member", "assignmentId": "a1"}))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("removing a gone assignment: status = %d, want 404 (%s)", rec.Code, rec.Body.String())
	}
}

func TestRenameRoleHandler(t *testing.T) {
	store := newAssignmentStore()
	store.primary["admin"] = orgtypes.RoleAdmin
	store.custom["Analyst"] = customRole("Analyst", PermOrgView)
	p := New(nil, store)

	rec := httptest.NewRecorder()
	p.RenameRoleHandler(rec, assignmentRequest(http.MethodPost, "/organizations/o1/roles/Analyst/rename",
		`{"name":"Lead Analyst"}`, map[string]string{"id": "o1", "name": "Analyst"}))
	if rec.Code != http.StatusOK {
		t.Fatalf("rename: status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	if _, ok := store.custom["Lead Analyst"]; !ok {
		t.Fatal("role not renamed")
	}

	rec = httptest.NewRecorder()
	p.RenameRoleHandler(rec, assignmentRequest(http.MethodPost, "/organizations/o1/roles/admin/rename",
		`{"name":"Boss"}`, map[string]string{"id": "o1", "name": "admin"}))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("renaming a compiled role: status = %d, want 400 (%s)", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	p.RenameRoleHandler(rec, assignmentRequest(http.MethodPost, "/organizations/o1/roles/Missing/rename",
		`{"name":"Whatever"}`, map[string]string{"id": "o1", "name": "Missing"}))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("renaming a missing role: status = %d, want 404 (%s)", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	p.RenameRoleHandler(rec, assignmentRequest(http.MethodPost, "/organizations/o1/roles/Lead%20Analyst/rename",
		`{"name":""}`, map[string]string{"id": "o1", "name": "Lead Analyst"}))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty new name: status = %d, want 400 (%s)", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	p.RenameRoleHandler(rec, assignmentRequest(http.MethodPost, "/organizations/o1/roles/Lead%20Analyst/rename",
		`{"name":"   "}`, map[string]string{"id": "o1", "name": "Lead Analyst"}))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("whitespace new name: status = %d, want 400 (%s)", rec.Code, rec.Body.String())
	}
}

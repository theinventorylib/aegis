package admin

import (
	"context"
	"testing"
	"time"

	"github.com/theinventorylib/aegis/auth"
	"github.com/theinventorylib/aegis/core"
	admintypes "github.com/theinventorylib/aegis/plugins/admin/types"
)

// mockStore implements admintypes.Store for testing without a database.
type mockStore struct {
	users  map[string]admintypes.User
	roles  map[string]string
	banned map[string]bool
}

func newMockStore() *mockStore {
	return &mockStore{
		users:  make(map[string]admintypes.User),
		roles:  make(map[string]string),
		banned: make(map[string]bool),
	}
}

func (m *mockStore) Create(_ context.Context, u admintypes.User) (admintypes.User, error) {
	m.users[u.ID] = u
	return u, nil
}

func (m *mockStore) GetByEmail(_ context.Context, email string) (admintypes.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return admintypes.User{}, core.NewAuthError(core.AuthErrorCodeUserNotFound, "not found")
}

func (m *mockStore) GetByID(_ context.Context, id string) (admintypes.User, error) {
	u, ok := m.users[id]
	if !ok {
		return admintypes.User{}, core.NewAuthError(core.AuthErrorCodeUserNotFound, "not found")
	}
	u.Role = m.roles[id]
	u.Banned = m.banned[id]
	return u, nil
}

func (m *mockStore) Update(_ context.Context, u admintypes.User) error {
	m.users[u.ID] = u
	return nil
}

func (m *mockStore) Delete(_ context.Context, id string) error {
	delete(m.users, id)
	return nil
}

func (m *mockStore) List(_ context.Context, offset, limit int) ([]admintypes.User, error) {
	users := make([]admintypes.User, 0, len(m.users))
	for _, u := range m.users {
		users = append(users, u)
	}
	if offset >= len(users) {
		return nil, nil
	}
	end := offset + limit
	if end > len(users) {
		end = len(users)
	}
	return users[offset:end], nil
}

func (m *mockStore) ListUsersRaw(_ context.Context, _, _ int) ([]map[string]any, error) {
	return nil, nil
}

func (m *mockStore) GetUserRaw(_ context.Context, _ string) (map[string]any, error) {
	return nil, nil
}

func (m *mockStore) Count(_ context.Context) (int, error) {
	return len(m.users), nil
}

func (m *mockStore) AssignRole(_ context.Context, userID string, role string) error {
	m.roles[userID] = role
	return nil
}

func (m *mockStore) RemoveRole(_ context.Context, userID string, _ string) error {
	delete(m.roles, userID)
	return nil
}

func (m *mockStore) GetRole(_ context.Context, userID string) (string, error) {
	return m.roles[userID], nil
}

func (m *mockStore) BanUser(_ context.Context, userID, _ string, _ *time.Time) error {
	m.banned[userID] = true
	u := m.users[userID]
	u.BanCounter++
	m.users[userID] = u
	return nil
}

func (m *mockStore) UnbanUser(_ context.Context, userID string) error {
	m.banned[userID] = false
	return nil
}

func (m *mockStore) GetStats(_ context.Context) (admintypes.StatsResponse, error) {
	return admintypes.StatsResponse{TotalUsers: len(m.users)}, nil
}

// ---- Test helpers ----

func newPlugin(store admintypes.Store) *Plugin {
	return &Plugin{store: store}
}

func seedUser(store *mockStore, email string) admintypes.User { //nolint:unparam // email is kept as a parameter for future flexibility
	u := admintypes.User{User: auth.User{ID: "u1", Email: email}}
	store.users["u1"] = u
	return u
}

// ---- Tests ----

func TestPlugin_AssignRole(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	p := newPlugin(store)
	seedUser(store, "user@example.com")

	if err := p.AssignRole(ctx, "u1", RoleAdmin); err != nil {
		t.Fatalf("AssignRole() error = %v", err)
	}

	role, err := store.GetRole(ctx, "u1")
	if err != nil {
		t.Fatalf("GetRole() error = %v", err)
	}
	if role != RoleAdmin {
		t.Errorf("role = %q, want %q", role, RoleAdmin)
	}
}

func TestPlugin_RemoveRole(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	p := newPlugin(store)
	seedUser(store, "user@example.com")
	_ = store.AssignRole(ctx, "u1", RoleAdmin)

	if err := p.RemoveRole(ctx, "u1", RoleAdmin); err != nil {
		t.Fatalf("RemoveRole() error = %v", err)
	}

	role, _ := store.GetRole(ctx, "u1")
	if role != "" {
		t.Errorf("expected empty role after removal, got %q", role)
	}
}

func TestPlugin_BanUnban(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	p := newPlugin(store)
	seedUser(store, "user@example.com")

	if err := p.BanUser(ctx, "u1", "Violation", nil); err != nil {
		t.Fatalf("BanUser() error = %v", err)
	}
	if !store.banned["u1"] {
		t.Error("expected user to be banned")
	}

	if err := p.UnbanUser(ctx, "u1"); err != nil {
		t.Fatalf("UnbanUser() error = %v", err)
	}
	if store.banned["u1"] {
		t.Error("expected user to be unbanned")
	}
}

func TestPlugin_EnrichUser_PopulatesBanFields(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	p := newPlugin(store)
	seedUser(store, "user@example.com")

	// Set up ban state directly in mock
	store.banned["u1"] = true
	u := store.users["u1"]
	u.BanReason = "spam"
	expiry := time.Now().Add(24 * time.Hour)
	u.BanExpiry = &expiry
	u.BanCounter = 2
	store.users["u1"] = u
	store.roles["u1"] = RoleAdmin

	enriched := core.NewEnrichedUser(&auth.User{ID: "u1", Email: "user@example.com"})

	if err := p.EnrichUser(ctx, enriched); err != nil {
		t.Fatalf("EnrichUser() error = %v", err)
	}

	if got := enriched.GetString(ExtKeyRole); got != RoleAdmin {
		t.Errorf("role = %q, want %q", got, RoleAdmin)
	}
	if got, ok := enriched.Get("banned").(bool); !ok || !got {
		t.Errorf("banned = %v, want true", enriched.Get("banned"))
	}
	if got := enriched.GetString("banReason"); got != "spam" {
		t.Errorf("banReason = %q, want %q", got, "spam")
	}
	if got, ok := enriched.Get("banCounter").(int); !ok || got != 2 {
		t.Errorf("banCounter = %v, want 2", enriched.Get("banCounter"))
	}
}

func TestPlugin_EnrichUser_MissingUser(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	p := newPlugin(store)

	enriched := core.NewEnrichedUser(&auth.User{ID: "nonexistent"})
	// Should return an error but the error does not affect enrichment of other fields
	err := p.EnrichUser(ctx, enriched)
	if err == nil {
		t.Error("expected error for nonexistent user")
	}
}

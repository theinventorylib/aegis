package admin

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/db"
	aegistesting "github.com/theinventorylib/aegis/testing"
)

// smartMockRow captures values and copies them to dest in Scan
type smartMockRow struct {
	values []interface{}
	err    error
}

func (r *smartMockRow) Scan(dest ...interface{}) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return fmt.Errorf("scan dest len %d != values len %d", len(dest), len(r.values))
	}
	for i, v := range r.values {
		if v == nil {
			continue
		}
		switch d := dest[i].(type) {
		case *string:
			if s, ok := v.(string); ok {
				*d = s
			}
		case *bool:
			if b, ok := v.(bool); ok {
				*d = b
			}
		case *int:
			if n, ok := v.(int); ok {
				*d = n
			}
		case *time.Time:
			if t, ok := v.(time.Time); ok {
				*d = t
			}
		case **time.Time:
			if v == nil {
				*d = nil
			} else if t, ok := v.(time.Time); ok {
				*d = &t
			}
		case *interface{}:
			*d = v
		default:
			// Handle other types or ignore
		}
	}
	return nil
}

func setupMockDB(_ *testing.T, mockDB *core.MockDB) {
	// We need a side store for admin fields since models.User doesn't have them
	adminFields := make(map[string]map[string]interface{})

	mockDB.ExecFunc = func(_ context.Context, query string, args ...interface{}) (db.Result, error) {
		if strings.Contains(query, "auth.user") {
			userID := args[0].(string)
			if _, ok := adminFields[userID]; !ok {
				adminFields[userID] = make(map[string]interface{})
			}

			if strings.Contains(query, "role =") {
				adminFields[userID]["role"] = args[1]
			}
			if strings.Contains(query, "banned = true") {
				adminFields[userID]["banned"] = true
				adminFields[userID]["ban_reason"] = args[1]
				adminFields[userID]["ban_expiry"] = args[2]
				count, _ := adminFields[userID]["ban_counter"].(int)
				adminFields[userID]["ban_counter"] = count + 1
			}
			if strings.Contains(query, "banned = false") {
				adminFields[userID]["banned"] = false
				adminFields[userID]["ban_reason"] = ""
				adminFields[userID]["ban_expiry"] = nil
			}
			return nil, nil
		}
		return nil, fmt.Errorf("unexpected exec query: %s", query)
	}

	mockDB.QueryRowFunc = func(ctx context.Context, query string, args ...interface{}) db.Row {
		if strings.Contains(query, "SELECT") && strings.Contains(query, "auth.user") {
			userID := args[0].(string)
			user, err := mockDB.GetUserByID(ctx, userID)
			if err != nil {
				return &smartMockRow{err: err}
			}

			fields := adminFields[userID]
			if fields == nil {
				fields = make(map[string]interface{})
			}

			role, _ := fields["role"].(string)
			if role == "" {
				role = "user"
			}
			banned, _ := fields["banned"].(bool)
			reason, _ := fields["ban_reason"].(string)
			expiry := fields["ban_expiry"]
			counter, _ := fields["ban_counter"].(int)

			// Order: id, created_at, updated_at, disabled, role, banned, reason, expiry, counter
			values := []interface{}{
				user.ID, user.CreatedAt, user.UpdatedAt, false,
				role, banned, reason, expiry, counter,
			}

			if strings.Contains(query, "COALESCE(role") && !strings.Contains(query, "banned") {
				return &smartMockRow{values: []interface{}{role}}
			}

			return &smartMockRow{values: values}
		}
		return &smartMockRow{err: fmt.Errorf("unexpected query row: %s", query)}
	}
}

func TestAdminPlugin_Metadata(t *testing.T) {
	plugin := New(nil)

	if plugin.Name() != "admin" {
		t.Errorf("Expected name 'admin', got '%s'", plugin.Name())
	}

	if plugin.Version() == "" {
		t.Error("Version should not be empty")
	}

	if plugin.Description() == "" {
		t.Error("Description should not be empty")
	}
}

func TestAdminPlugin_Migrations(t *testing.T) {
	plugin := New(nil)
	migrations := plugin.GetMigrations()

	if len(migrations) == 0 {
		t.Fatal("Admin plugin should provide migrations")
	}

	// Verify migration includes RBAC and ban fields
	firstMigration := migrations[0]
	if firstMigration.Version != "001" {
		t.Errorf("Expected version '001', got '%s'", firstMigration.Version)
	}

	// Check that migration includes expected fields
	expectedFields := []string{"role", "banned", "ban_reason", "ban_expiry", "ban_counter"}
	for _, field := range expectedFields {
		if !contains(firstMigration.Up, field) {
			t.Errorf("Migration Up should contain field '%s'", field)
		}
		if !contains(firstMigration.Down, field) {
			t.Errorf("Migration Down should contain field '%s'", field)
		}
	}

	t.Logf("✓ Admin plugin provides %d migration(s) with RBAC and ban fields", len(migrations))
}

func TestAdminPlugin_RequiresAdminRole(t *testing.T) {
	testAegis := aegistesting.Setup(t)
	defer testAegis.Cleanup()
	setupMockDB(t, testAegis.DB)

	plugin := New(testAegis.DB)
	adminDB := NewDB(testAegis.DB)
	ctx := context.Background()

	if err := plugin.Init(ctx, testAegis.Aegis); err != nil {
		t.Fatalf("Plugin Init failed: %v", err)
	}
	plugin.MountRoutes(testAegis.Router, "/api")

	regularUser := testAegis.CreateTestUser(t, "user-1")
	if err := adminDB.SetUserRole(ctx, regularUser.ID, "user"); err != nil {
		t.Fatalf("Failed to set user role: %v", err)
	}

	session := testAegis.CreateTestSession(t, regularUser.ID)
	rec := testAegis.AuthenticatedRequest(t, "GET", "/api/admin/users", session.Token)

	if rec.Code != 403 {
		t.Errorf("Expected 403 Forbidden for non-admin user, got %d", rec.Code)
	}
	t.Log("✓ Non-admin users are correctly denied access")
}

func TestAdminPlugin_AdminAccess(t *testing.T) {
	testAegis := aegistesting.Setup(t)
	defer testAegis.Cleanup()
	setupMockDB(t, testAegis.DB)

	plugin := New(testAegis.DB)
	adminDB := NewDB(testAegis.DB)
	ctx := context.Background()

	if err := plugin.Init(ctx, testAegis.Aegis); err != nil {
		t.Fatalf("Plugin Init failed: %v", err)
	}
	plugin.MountRoutes(testAegis.Router, "/api")

	adminUser := testAegis.CreateTestUser(t, "admin-1")
	if err := adminDB.SetUserRole(ctx, adminUser.ID, "admin"); err != nil {
		t.Fatalf("Failed to set admin role: %v", err)
	}

	session := testAegis.CreateTestSession(t, adminUser.ID)

	// We need to mock ListUsersRaw for the handler
	testAegis.DB.QueryFunc = func(_ context.Context, _ string, _ ...interface{}) (db.Rows, error) {
		// Just return empty rows for list
		return &mockRows{}, nil
	}

	rec := testAegis.AuthenticatedRequest(t, "GET", "/api/admin/users", session.Token)

	if rec.Code != 200 {
		t.Errorf("Expected 200 OK for admin user, got %d", rec.Code)
	}
	t.Log("✓ Admin users can successfully access admin endpoints")
}

func TestAdminPlugin_BanUser(t *testing.T) {
	testAegis := aegistesting.Setup(t)
	defer testAegis.Cleanup()
	setupMockDB(t, testAegis.DB)

	adminDB := NewDB(testAegis.DB)
	ctx := context.Background()

	user := testAegis.CreateTestUser(t, "test-user")

	if err := adminDB.BanUser(ctx, user.ID, "Test ban", nil); err != nil {
		t.Fatalf("Failed to ban user: %v", err)
	}

	bannedUser, err := adminDB.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve banned user: %v", err)
	}

	if !bannedUser.Banned {
		t.Error("User should be banned")
	}
	if bannedUser.BanReason != "Test ban" {
		t.Errorf("Expected ban reason 'Test ban', got '%s'", bannedUser.BanReason)
	}
	if bannedUser.BanCounter != 1 {
		t.Errorf("Expected ban counter 1, got %d", bannedUser.BanCounter)
	}
	t.Log("✓ User ban applied successfully with reason and counter")
}

func TestAdminPlugin_BanWithExpiry(t *testing.T) {
	testAegis := aegistesting.Setup(t)
	defer testAegis.Cleanup()
	setupMockDB(t, testAegis.DB)

	adminDB := NewDB(testAegis.DB)
	ctx := context.Background()

	user := testAegis.CreateTestUser(t, "test-user")
	expiry := time.Now().Add(24 * time.Hour)

	if err := adminDB.BanUser(ctx, user.ID, "Temporary ban", expiry); err != nil {
		t.Fatalf("Failed to ban user: %v", err)
	}

	bannedUser, err := adminDB.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve banned user: %v", err)
	}

	if !bannedUser.Banned {
		t.Error("User should be banned")
	}
	if bannedUser.BanExpiry == nil {
		t.Error("Ban expiry should be set")
	}
	t.Log("✓ Temporary ban with expiry applied successfully")
}

func TestAdminPlugin_UnbanUser(t *testing.T) {
	testAegis := aegistesting.Setup(t)
	defer testAegis.Cleanup()
	setupMockDB(t, testAegis.DB)

	adminDB := NewDB(testAegis.DB)
	ctx := context.Background()

	user := testAegis.CreateTestUser(t, "test-user")

	_ = adminDB.BanUser(ctx, user.ID, "First ban", nil)
	_ = adminDB.UnbanUser(ctx, user.ID)
	_ = adminDB.BanUser(ctx, user.ID, "Second ban", nil)

	bannedUser, err := adminDB.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to get user before unban: %v", err)
	}
	expectedCounter := bannedUser.BanCounter

	if err := adminDB.UnbanUser(ctx, user.ID); err != nil {
		t.Fatalf("Failed to unban user: %v", err)
	}

	unbannedUser, err := adminDB.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve unbanned user: %v", err)
	}

	if unbannedUser.Banned {
		t.Error("User should not be banned")
	}
	if unbannedUser.BanReason != "" {
		t.Error("Ban reason should be cleared")
	}
	if unbannedUser.BanCounter != expectedCounter {
		t.Errorf("Ban counter should be preserved at %d, got %d", expectedCounter, unbannedUser.BanCounter)
	}
	t.Log("✓ User unbanned successfully, ban counter preserved")
}

func TestAdminPlugin_BanCounterIncrement(t *testing.T) {
	testAegis := aegistesting.Setup(t)
	defer testAegis.Cleanup()
	setupMockDB(t, testAegis.DB)

	adminDB := NewDB(testAegis.DB)
	ctx := context.Background()

	user := testAegis.CreateTestUser(t, "test-user")

	for i := 1; i <= 3; i++ {
		_ = adminDB.BanUser(ctx, user.ID, "Ban iteration", nil)
		_ = adminDB.UnbanUser(ctx, user.ID)
	}

	finalUser, err := adminDB.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if finalUser.BanCounter != 3 {
		t.Errorf("Expected ban counter 3, got %d", finalUser.BanCounter)
	}
	t.Log("✓ Ban counter increments correctly across multiple bans")
}

// mockRows implements db.Rows
type mockRows struct{}

func (r *mockRows) Next() bool                  { return false }
func (r *mockRows) Scan(_ ...interface{}) error { return nil }
func (r *mockRows) Close()                      {}
func (r *mockRows) Err() error                  { return nil }

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

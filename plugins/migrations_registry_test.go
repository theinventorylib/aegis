package plugins

import (
	"testing"
)

func TestScanMigration_IFNOTEXISTS(t *testing.T) {
	// Test SQL with IF NOT EXISTS
	sql := `
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS email_verified INTEGER NOT NULL DEFAULT 0;
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS role TEXT DEFAULT 'user';
ALTER TABLE "user" ADD COLUMN phone_number TEXT;
`

	tables, columns := scanMigration(sql)

	// Should not capture any tables
	if len(tables) != 0 {
		t.Errorf("expected 0 tables, got %d: %v", len(tables), tables)
	}

	// Should capture 3 columns with correct names
	expected := []string{"user.email_verified", "user.role", "user.phone_number"}
	if len(columns) != len(expected) {
		t.Errorf("expected %d columns, got %d: %v", len(expected), len(columns), columns)
	}

	for i, exp := range expected {
		if i >= len(columns) {
			break
		}
		if columns[i] != exp {
			t.Errorf("column %d: expected %q, got %q", i, exp, columns[i])
		}
	}

	// Most importantly: should NOT have "user.if"
	for _, col := range columns {
		if col == "user.if" {
			t.Error("regex incorrectly captured 'if' as column name - IF NOT EXISTS not handled properly")
		}
	}
}

func TestMigrationRegistry_IFNOTEXISTS(t *testing.T) {
	mr := NewMigrationRegistry()

	// Register admin plugin migrations (uses IF NOT EXISTS)
	adminMigrations := []Migration{
		{
			Version:     1,
			Description: "initial",
			Up:          `ALTER TABLE "user" ADD COLUMN IF NOT EXISTS role TEXT DEFAULT 'user';`,
		},
	}

	if err := mr.Register("admin", adminMigrations); err != nil {
		t.Fatalf("failed to register admin: %v", err)
	}

	// Register emailotp plugin migrations (also uses IF NOT EXISTS)
	emailMigrations := []Migration{
		{
			Version:     1,
			Description: "initial",
			Up:          `ALTER TABLE "user" ADD COLUMN IF NOT EXISTS email_verified INTEGER NOT NULL DEFAULT 0;`,
		},
	}

	// This should succeed - they're adding different columns
	if err := mr.Register("email-otp", emailMigrations); err != nil {
		t.Errorf("failed to register email-otp: %v", err)
	}

	// Verify ownership
	if owner := mr.columns["user.role"]; owner != "admin" {
		t.Errorf("expected user.role owner to be 'admin', got %q", owner)
	}
	if owner := mr.columns["user.email_verified"]; owner != "email-otp" {
		t.Errorf("expected user.email_verified owner to be 'email-otp', got %q", owner)
	}
}

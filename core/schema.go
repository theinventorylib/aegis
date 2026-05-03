package core

import (
	"context"
	"database/sql"
	"fmt"
)

// SchemaValidator provides database schema validation for Aegis.
//
// This validator helps ensure that the required database tables and columns
// exist before the application starts. It's used by:
//   - Core Aegis (validates user, accounts, session, verification tables)
//   - Plugins (each plugin can validate its own schema requirements)
//
// The validator executes SQL queries to check for table/column existence,
// collecting all validation errors before failing. This provides complete
// error visibility instead of failing on the first missing table.
//
// Example:
//
//	validator := core.NewSchemaValidator(db)
//	err := validator.ValidateRequirements(ctx, core.SchemaRequirements())
//	if err != nil {
//		log.Fatal("Database schema validation failed:", err)
//	}
type SchemaValidator struct {
	db *sql.DB
}

// SchemaRequirement defines a single schema validation check.
//
// Each requirement represents one validation rule (table exists, column exists,
// index exists, etc.). The validator executes the Query and expects it to
// succeed without error for the requirement to pass.
type SchemaRequirement struct {
	// Name is a human-readable identifier for this requirement
	// Example: "Table 'users' exists", "Column 'accounts.password_hash' exists"
	Name string

	// Schema is the database schema name (optional, for multi-schema databases)
	Schema string

	// Table is the table name being validated (optional, for documentation)
	Table string

	// Query is the SQL query to execute
	// Should succeed without error if the requirement is met
	// Example: "SELECT 1 FROM users WHERE 1=0"
	Query string

	// Description explains what the validation failure means
	// This is shown in the error message if the query returns no rows
	// Example: "Table 'users' does not exist or is empty"
	Description string
}

// NewSchemaValidator creates a new schema validator for the given database.
//
// The validator will execute queries against this database connection to
// validate schema requirements.
func NewSchemaValidator(db *sql.DB) *SchemaValidator {
	return &SchemaValidator{db: db}
}

// ValidateRequirements validates a list of schema requirements.
//
// This method runs all validations and collects ALL errors before returning.
// This provides complete visibility into schema problems instead of failing
// on the first error.
//
// Returns nil if all requirements pass, or an error listing all failures.
//
// Example:
//
//	requirements := []core.SchemaRequirement{
//		core.ValidateTableExists("users"),
//		core.ValidateColumnExists("users", "email"),
//		core.ValidateColumnExists("users", "password_hash"),
//	}
//	err := validator.ValidateRequirements(ctx, requirements)
func (v *SchemaValidator) ValidateRequirements(ctx context.Context, requirements []SchemaRequirement) error {
	var errors []error

	for _, req := range requirements {
		if err := v.validateRequirement(ctx, req); err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", req.Name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("schema validation failed with %d errors: %v", len(errors), errors)
	}

	return nil
}

// validateRequirement validates a single schema requirement.
//
// Executes the requirement's Query and checks if it succeeds without error.
// If the query fails, the requirement is considered failed.
func (v *SchemaValidator) validateRequirement(ctx context.Context, req SchemaRequirement) (err error) {
	rows, qerr := v.db.QueryContext(ctx, req.Query)
	if qerr != nil {
		return fmt.Errorf("query failed: %w", qerr)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("rows close error: %w", cerr)
		}
	}()

	// If the query returns no rows, treat the requirement as failed.
	// This covers information_schema checks which return zero rows when
	// the table/column is missing (instead of producing a SQL error).
	if !rows.Next() {
		if ierr := rows.Err(); ierr != nil {
			return fmt.Errorf("query iteration error: %w", ierr)
		}
		if req.Description != "" {
			return fmt.Errorf("%s", req.Description)
		}
		return fmt.Errorf("requirement not satisfied")
	}

	return nil
}

// ValidateTableExists creates a requirement to check if a table exists.
//
// tableName must be a valid SQL identifier ([A-Za-z_][A-Za-z0-9_]*).
// This panics on malformed identifiers to fail fast at startup —
// see SanitizeSQLIdentifier for rationale.
func ValidateTableExists(tableName string) SchemaRequirement {
	tableName = SanitizeSQLIdentifier(tableName)
	return SchemaRequirement{
		Name: fmt.Sprintf("Table '%s' exists", tableName),
		Query: fmt.Sprintf(
			"SELECT table_name FROM information_schema.tables WHERE table_name = '%s' LIMIT 1",
			tableName,
		),
		Description: fmt.Sprintf("Table '%s' does not exist", tableName),
	}
}

// ValidateColumnExists creates a requirement to check if a column exists in a table.
//
// This is the simplest column check: presence only. If you also need to
// detect schema drift (wrong type, unexpected NULL/NOT NULL, etc.), use
// ValidateColumnSpec instead — silent type changes have caused real
// production bugs in this codebase before.
//
// The generated query targets information_schema, which is supported by
// PostgreSQL and MySQL but not by SQLite. SQLite callers should use
// ValidateColumnExistsForDialect with DialectSQLite (see schema_dialects.go).
func ValidateColumnExists(tableName, columnName string) SchemaRequirement {
	tableName = SanitizeSQLIdentifier(tableName)
	columnName = SanitizeSQLIdentifier(columnName)
	return SchemaRequirement{
		Name: fmt.Sprintf("Column '%s.%s' exists", tableName, columnName),
		Query: fmt.Sprintf(
			"SELECT column_name FROM information_schema.columns WHERE table_name = '%s' AND column_name = '%s' LIMIT 1",
			tableName, columnName,
		),
		Description: fmt.Sprintf("Column '%s' does not exist in table '%s'", columnName, tableName),
	}
}

// ColumnSpec describes the expected shape of a column for validation.
//
// Either field may be left at its zero value to skip that part of the
// check:
//   - DataType == "": don't compare types (presence-only check).
//   - Nullable == nil: don't compare nullability.
//
// DataType is matched against information_schema.columns.data_type using
// a case-insensitive equality test. Database-specific type aliases (e.g.
// "int" vs "integer", "varchar" vs "character varying") are NOT
// normalised — pass the canonical name your dialect reports. SQLite is
// not supported by information_schema and will fall back to a
// presence-only check.
type ColumnSpec struct {
	DataType string
	Nullable *bool
}

// BoolPtr is a tiny convenience for building ColumnSpec.Nullable inline.
func BoolPtr(b bool) *bool { return &b }

// ValidateColumnSpec creates a requirement that checks not just the
// existence of a column, but also (optionally) its data type and
// nullability. It is the recommended replacement for
// ValidateColumnExists when you care about detecting schema drift.
//
// The query selects the row from information_schema.columns and uses
// boolean predicates to enforce the spec. A failed match returns zero
// rows, which the validator interprets as a failure with a descriptive
// message.
func ValidateColumnSpec(tableName, columnName string, spec ColumnSpec) SchemaRequirement {
	tableName = SanitizeSQLIdentifier(tableName)
	columnName = SanitizeSQLIdentifier(columnName)
	conditions := []string{
		fmt.Sprintf("table_name = '%s'", tableName),
		fmt.Sprintf("column_name = '%s'", columnName),
	}
	descParts := []string{fmt.Sprintf("column '%s.%s'", tableName, columnName)}

	if spec.DataType != "" {
		dt := sanitizeSQLLiteral(spec.DataType)
		conditions = append(conditions, fmt.Sprintf("LOWER(data_type) = LOWER('%s')", dt))
		descParts = append(descParts, fmt.Sprintf("data_type=%s", spec.DataType))
	}
	if spec.Nullable != nil {
		want := "YES"
		if !*spec.Nullable {
			want = "NO"
		}
		conditions = append(conditions, fmt.Sprintf("UPPER(is_nullable) = '%s'", want))
		descParts = append(descParts, fmt.Sprintf("nullable=%v", *spec.Nullable))
	}

	return SchemaRequirement{
		Name: fmt.Sprintf("Column spec '%s.%s' matches", tableName, columnName),
		Query: fmt.Sprintf(
			"SELECT column_name FROM information_schema.columns WHERE %s LIMIT 1",
			joinAnd(conditions),
		),
		Description: fmt.Sprintf("expected %s — column missing or schema drift detected", joinComma(descParts)),
	}
}

// joinAnd / joinComma are tiny local helpers to avoid pulling strings into
// this file for two call sites.
func joinAnd(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " AND "
		}
		out += p
	}
	return out
}

func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}

// SchemaRequirements returns the schema requirements for core Aegis tables.
//
// This validates the existence of:
//   - Tables: user, accounts, verification, session
//   - All required columns in each table
func SchemaRequirements() []SchemaRequirement {
	return []SchemaRequirement{
		ValidateTableExists("user"),
		ValidateTableExists("accounts"),
		ValidateTableExists("verification"),
		ValidateTableExists("session"),

		ValidateColumnExists("user", "id"),
		ValidateColumnExists("user", "avatar"),
		ValidateColumnExists("user", "name"),
		ValidateColumnExists("user", "email"),
		ValidateColumnExists("user", "created_at"),
		ValidateColumnExists("user", "updated_at"),
		ValidateColumnExists("user", "disabled"),

		ValidateColumnExists("accounts", "id"),
		ValidateColumnExists("accounts", "user_id"),
		ValidateColumnExists("accounts", "provider"),
		ValidateColumnExists("accounts", "provider_account_id"),
		ValidateColumnExists("accounts", "password_hash"),
		ValidateColumnExists("accounts", "access_token"),
		ValidateColumnExists("accounts", "refresh_token"),
		ValidateColumnExists("accounts", "expires_at"),
		ValidateColumnExists("accounts", "created_at"),
		ValidateColumnExists("accounts", "updated_at"),

		ValidateColumnExists("verification", "id"),
		ValidateColumnExists("verification", "identifier"),
		ValidateColumnExists("verification", "token"),
		ValidateColumnExists("verification", "type"),
		ValidateColumnExists("verification", "expires_at"),
		ValidateColumnExists("verification", "created_at"),

		ValidateColumnExists("session", "id"),
		ValidateColumnExists("session", "user_id"),
		ValidateColumnExists("session", "token"),
		ValidateColumnExists("session", "refresh_token"),
		ValidateColumnExists("session", "expires_at"),
		ValidateColumnExists("session", "created_at"),
		ValidateColumnExists("session", "ip_address"),
		ValidateColumnExists("session", "user_agent"),
	}
}

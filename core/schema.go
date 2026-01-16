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
// This queries `information_schema.tables` for the given table name.
// The actual query generated is:
//
//	SELECT table_name FROM information_schema.tables WHERE table_name = '<name>' LIMIT 1
//
// Example:
//
//	requirement := core.ValidateTableExists("users")
//	// Will check: SELECT table_name FROM information_schema.tables WHERE table_name = 'users' LIMIT 1
func ValidateTableExists(tableName string) SchemaRequirement {
	return SchemaRequirement{
		Name:        fmt.Sprintf("Table '%s' exists", tableName),
		Query:       fmt.Sprintf("SELECT table_name FROM information_schema.tables WHERE table_name = '%s' LIMIT 1", tableName),
		Description: fmt.Sprintf("Table '%s' does not exist", tableName),
	}
}

// ValidateColumnExists creates a requirement to check if a column exists in a table.
//
// This queries `information_schema.columns` for the given table and column.
// The actual query generated is:
//
//	SELECT column_name FROM information_schema.columns WHERE table_name = '<table>' AND column_name = '<column>' LIMIT 1
//
// Example:
//
//	requirement := core.ValidateColumnExists("users", "email")
//	// Will check: SELECT column_name FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'email' LIMIT 1
func ValidateColumnExists(tableName, columnName string) SchemaRequirement {
	return SchemaRequirement{
		Name:        fmt.Sprintf("Column '%s.%s' exists", tableName, columnName),
		Query:       fmt.Sprintf("SELECT column_name FROM information_schema.columns WHERE table_name = '%s' AND column_name = '%s' LIMIT 1", tableName, columnName),
		Description: fmt.Sprintf("Column '%s' does not exist in table '%s'", columnName, tableName),
	}
}

// SchemaRequirements returns the schema requirements for core Aegis tables.
//
// This validates the existence of:
//   - Tables: user, accounts, verification, session
//   - All required columns in each table
//
// Call this during application startup to ensure the database is properly migrated:
//
//	validator := core.NewSchemaValidator(db)
//	if err := validator.ValidateRequirements(ctx, core.SchemaRequirements()); err != nil {
//		log.Fatal("Schema validation failed:", err)
//	}
func SchemaRequirements() []SchemaRequirement {
	return []SchemaRequirement{
		ValidateTableExists("user"),
		ValidateTableExists("accounts"),
		ValidateTableExists("verification"),
		ValidateTableExists("session"),

		ValidateColumnExists("user", "id"),
		ValidateColumnExists("user", "name"),
		ValidateColumnExists("user", "email"),
		ValidateColumnExists("user", "created_at"),
		ValidateColumnExists("user", "updated_at"),
		ValidateColumnExists("user", "disabled"),

		ValidateColumnExists("accounts", "id"),
		ValidateColumnExists("accounts", "user_id"),
		ValidateColumnExists("accounts", "provider"),
		ValidateColumnExists("accounts", "password_hash"),
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
		ValidateColumnExists("session", "expires_at"),
		ValidateColumnExists("session", "created_at"),
	}
}

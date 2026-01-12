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
func (v *SchemaValidator) validateRequirement(ctx context.Context, req SchemaRequirement) error {
	rows, err := v.db.QueryContext(ctx, req.Query)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer func() { err := rows.Close(); _ = err }()

	return nil
}

// ValidateTableExists creates a requirement to check if a table exists.
//
// This generates a "SELECT 1 FROM table WHERE 1=0" query that will fail
// if the table doesn't exist.
//
// Example:
//
//	requirement := core.ValidateTableExists("users")
//	// Will check: SELECT 1 FROM users WHERE 1=0
func ValidateTableExists(tableName string) SchemaRequirement {
	return SchemaRequirement{
		Name:        fmt.Sprintf("Table '%s' exists", tableName),
		Query:       fmt.Sprintf("SELECT 1 FROM %s WHERE 1=0", tableName),
		Description: fmt.Sprintf("Table '%s' does not exist", tableName),
	}
}

// ValidateColumnExists creates a requirement to check if a column exists in a table.
//
// This generates a "SELECT column FROM table WHERE 1=0" query that will fail
// if the column doesn't exist.
//
// Example:
//
//	requirement := core.ValidateColumnExists("users", "email")
//	// Will check: SELECT email FROM users WHERE 1=0
func ValidateColumnExists(tableName, columnName string) SchemaRequirement {
	return SchemaRequirement{
		Name:        fmt.Sprintf("Column '%s.%s' exists", tableName, columnName),
		Query:       fmt.Sprintf("SELECT %s FROM %s WHERE 1=0", columnName, tableName),
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

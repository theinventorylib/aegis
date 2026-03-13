package admin

import (
	_ "embed"
	"fmt"

	"github.com/theinventorylib/aegis/plugins"
)

//go:embed migrations/postgres/001_initial.up.sql
var postgresSchema string

//go:embed migrations/mysql/001_initial.up.sql
var mysqlSchema string

//go:embed migrations/sqlite/001_initial.up.sql
var sqliteSchema string

// GetSchema returns the database schema for the admin plugin.
//
// The schema extends the 'user' table with admin-specific columns:
//   - role (VARCHAR): User role for RBAC (e.g., "admin", "moderator")
//   - banned (BOOLEAN): Ban status (true if user is banned)
//   - ban_reason (TEXT): Admin-provided reason for ban
//   - ban_expiry (TIMESTAMP, nullable): Ban expiration date (NULL for permanent)
//   - ban_counter (INTEGER): Number of times user has been banned
//
// These extensions enable role-based authorization and ban management.
//
// Parameters:
//   - dialect: Database dialect (postgres, mysql)
//
// Returns:
//   - *plugins.Schema: Schema definition with SQL DDL
//   - error: If dialect is not supported
func GetSchema(dialect plugins.Dialect) (*plugins.Schema, error) {
	var sql string

	switch dialect {
	case plugins.DialectPostgres:
		sql = postgresSchema
	case plugins.DialectMySQL:
		sql = mysqlSchema
	case plugins.DialectSQLite:
		sql = sqliteSchema
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dialect)
	}

	info := plugins.SchemaInfo{
		Package:     "github.com/theinventorylib/aegis/plugins/admin",
		Version:     1,
		Description: "Admin plugin schema",
	}

	return &plugins.Schema{
		Dialect: dialect,
		SQL:     sql,
		Info:    info,
	}, nil
}

// GetSchemaRequirements returns schema validation requirements for the admin plugin.
//
// This function defines structural requirements that must be satisfied for the plugin
// to function correctly. The Init() method validates these requirements at startup.
//
// Validation Checks:
//   - Column existence: role, banned, ban_reason, ban_expiry, ban_counter in 'user' table
//   - Column properties: Data types, nullability (not fully implemented yet)
//
// These checks help detect schema drift, incomplete migrations, or manual schema changes
// that could break admin functionality.
//
// Parameters:
//   - dialect: Database dialect (postgres, mysql)
//
// Returns:
//   - []plugins.SchemaRequirement: List of validation requirements
func GetSchemaRequirements(dialect plugins.Dialect) []plugins.SchemaRequirement {
	switch dialect {
	case plugins.DialectPostgres, plugins.DialectMySQL:
		return []plugins.SchemaRequirement{
			// Column existence and properties
			plugins.ValidateColumnExists("user", "role"),
			plugins.ValidateColumnExists("user", "banned"),
			plugins.ValidateColumnExists("user", "ban_reason"),
			plugins.ValidateColumnExists("user", "ban_expiry"),
			plugins.ValidateColumnExists("user", "ban_counter"),
		}
	default:
		return []plugins.SchemaRequirement{}
	}
}

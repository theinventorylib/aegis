package admin

import (
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/plugins"
)

// GetSchemaRequirements returns schema validation requirements for the admin plugin.
//
// This function defines structural requirements that must be satisfied for the plugin
// to function correctly. The Init() method validates these requirements at startup.
//
// Validation Checks:
//   - Column existence: role, banned, ban_reason, ban_expiry, ban_counter in 'user' table
//   - Security-critical columns (role, banned) are checked with ColumnSpec to detect
//     type/nullability drift in addition to presence.
//
// These checks help detect schema drift, incomplete migrations, or manual schema changes
// that could break admin functionality.
//
// Parameters:
//   - dialect: Database dialect (postgres, mysql, sqlite)
//
// Returns:
//   - []plugins.SchemaRequirement: List of validation requirements
func GetSchemaRequirements(dialect plugins.Dialect) []plugins.SchemaRequirement {
	d := string(dialect)
	switch dialect {
	case plugins.DialectPostgres, plugins.DialectMySQL, plugins.DialectSQLite:
		// role default type for postgres/mysql is varchar; sqlite stores
		// it as TEXT. We therefore only spec nullability (NOT NULL) and
		// leave the type unconstrained, since SQLite type affinity does
		// not match information_schema's data_type strings 1:1.
		notNull := core.BoolPtr(false)
		return []plugins.SchemaRequirement{
			plugins.ValidateColumnSpecForDialect(d, "user", "role", core.ColumnSpec{Nullable: notNull}),
			plugins.ValidateColumnSpecForDialect(d, "user", "banned", core.ColumnSpec{Nullable: notNull}),
			plugins.ValidateColumnExistsForDialect(d, "user", "ban_reason"),
			plugins.ValidateColumnExistsForDialect(d, "user", "ban_expiry"),
			plugins.ValidateColumnExistsForDialect(d, "user", "ban_counter"),
		}
	default:
		return []plugins.SchemaRequirement{}
	}
}

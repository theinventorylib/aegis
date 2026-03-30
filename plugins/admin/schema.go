package admin

import (
	"github.com/theinventorylib/aegis/plugins"
)

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

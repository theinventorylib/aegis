package organizations

import (
	"github.com/theinventorylib/aegis/plugins"
)

// GetSchemaRequirements returns schema validation requirements for the organizations plugin.
//
// This function defines structural requirements that must be satisfied for the plugin
// to function correctly. The Init() method validates these requirements at startup.
//
// Validation Checks:
//   - Table existence: organization, members, team, team_member
//   - Column existence: All required columns in each table
//   - Column properties: Data types, nullability (not implemented yet)
//
// These checks help detect schema drift, incomplete migrations, or manual schema changes.
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
			// Table existence
			plugins.ValidateTableExists("organization"),
			plugins.ValidateTableExists("members"),
			plugins.ValidateTableExists("team"),
			plugins.ValidateTableExists("team_member"),
			// Column existence and properties
			plugins.ValidateColumnExists("organization", "id"),
			plugins.ValidateColumnExists("organization", "name"),
			plugins.ValidateColumnExists("organization", "slug"),
			plugins.ValidateColumnExists("organization", "disabled"),
			plugins.ValidateColumnExists("organization", "created_at"),
			plugins.ValidateColumnExists("organization", "updated_at"),
			plugins.ValidateColumnExists("members", "id"),
			plugins.ValidateColumnExists("members", "user_id"),
			plugins.ValidateColumnExists("members", "organization_id"),
			plugins.ValidateColumnExists("members", "role"),
			plugins.ValidateColumnExists("members", "created_at"),
			plugins.ValidateColumnExists("members", "updated_at"),
			plugins.ValidateColumnExists("team", "id"),
			plugins.ValidateColumnExists("team", "organization_id"),
			plugins.ValidateColumnExists("team", "name"),
			plugins.ValidateColumnExists("team", "description"),
			plugins.ValidateColumnExists("team", "created_at"),
			plugins.ValidateColumnExists("team", "updated_at"),
			plugins.ValidateColumnExists("team_member", "id"),
			plugins.ValidateColumnExists("team_member", "team_id"),
			plugins.ValidateColumnExists("team_member", "user_id"),
			plugins.ValidateColumnExists("team_member", "role"),
			plugins.ValidateColumnExists("team_member", "created_at"),
			plugins.ValidateColumnExists("team_member", "updated_at"),
		}
	default:
		return []plugins.SchemaRequirement{}
	}
}

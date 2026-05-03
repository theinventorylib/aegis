package organizations

import (
	"github.com/theinventorylib/aegis/core"
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
//   - Security-critical role columns are checked with ColumnSpec to detect
//     type/nullability drift that could cause silent privilege confusion.
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
		notNull := core.BoolPtr(false)
		return []plugins.SchemaRequirement{
			plugins.ValidateTableExistsForDialect(d, "organization"),
			plugins.ValidateTableExistsForDialect(d, "members"),
			plugins.ValidateTableExistsForDialect(d, "team"),
			plugins.ValidateTableExistsForDialect(d, "team_member"),
			plugins.ValidateColumnExistsForDialect(d, "organization", "id"),
			plugins.ValidateColumnExistsForDialect(d, "organization", "name"),
			plugins.ValidateColumnExistsForDialect(d, "organization", "slug"),
			plugins.ValidateColumnExistsForDialect(d, "organization", "disabled"),
			plugins.ValidateColumnExistsForDialect(d, "organization", "created_at"),
			plugins.ValidateColumnExistsForDialect(d, "organization", "updated_at"),
			plugins.ValidateColumnExistsForDialect(d, "members", "id"),
			plugins.ValidateColumnSpecForDialect(d, "members", "user_id", core.ColumnSpec{Nullable: notNull}),
			plugins.ValidateColumnSpecForDialect(d, "members", "organization_id", core.ColumnSpec{Nullable: notNull}),
			plugins.ValidateColumnSpecForDialect(d, "members", "role", core.ColumnSpec{Nullable: notNull}),
			plugins.ValidateColumnExistsForDialect(d, "members", "created_at"),
			plugins.ValidateColumnExistsForDialect(d, "members", "updated_at"),
			plugins.ValidateColumnExistsForDialect(d, "team", "id"),
			plugins.ValidateColumnSpecForDialect(d, "team", "organization_id", core.ColumnSpec{Nullable: notNull}),
			plugins.ValidateColumnExistsForDialect(d, "team", "name"),
			plugins.ValidateColumnExistsForDialect(d, "team", "description"),
			plugins.ValidateColumnExistsForDialect(d, "team", "created_at"),
			plugins.ValidateColumnExistsForDialect(d, "team", "updated_at"),
			plugins.ValidateColumnExistsForDialect(d, "team_member", "id"),
			plugins.ValidateColumnSpecForDialect(d, "team_member", "team_id", core.ColumnSpec{Nullable: notNull}),
			plugins.ValidateColumnSpecForDialect(d, "team_member", "user_id", core.ColumnSpec{Nullable: notNull}),
			plugins.ValidateColumnSpecForDialect(d, "team_member", "role", core.ColumnSpec{Nullable: notNull}),
			plugins.ValidateColumnExistsForDialect(d, "team_member", "created_at"),
			plugins.ValidateColumnExistsForDialect(d, "team_member", "updated_at"),
		}
	default:
		return []plugins.SchemaRequirement{}
	}
}

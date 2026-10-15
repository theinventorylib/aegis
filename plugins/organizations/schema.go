package organizations

import (
	"github.com/theinventorylib/aegis/v2/core"
	"github.com/theinventorylib/aegis/v2/plugins"
)

// GetSchemaRequirements returns schema validation requirements for the organizations plugin.
//
// This function defines structural requirements that must be satisfied for the plugin
// to function correctly. The Init() method validates these requirements at startup.
//
// Validation Checks:
//   - Table existence: organization, members, team, team_member, invitation,
//     member_role_assignment
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
		notNull := new(false)
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
			plugins.ValidateTableExistsForDialect(d, "invitation"),
			plugins.ValidateColumnExistsForDialect(d, "invitation", "id"),
			plugins.ValidateColumnSpecForDialect(d, "invitation", "organization_id", core.ColumnSpec{Nullable: notNull}),
			plugins.ValidateColumnExistsForDialect(d, "invitation", "team_id"),
			plugins.ValidateColumnExistsForDialect(d, "invitation", "email"),
			plugins.ValidateColumnSpecForDialect(d, "invitation", "role", core.ColumnSpec{Nullable: notNull}),
			plugins.ValidateColumnSpecForDialect(d, "invitation", "inviter_id", core.ColumnSpec{Nullable: notNull}),
			plugins.ValidateColumnExistsForDialect(d, "invitation", "token_hash"),
			plugins.ValidateColumnExistsForDialect(d, "invitation", "status"),
			plugins.ValidateColumnExistsForDialect(d, "invitation", "expires_at"),
			plugins.ValidateColumnExistsForDialect(d, "invitation", "created_at"),
			plugins.ValidateColumnExistsForDialect(d, "invitation", "updated_at"),
			plugins.ValidateTableExistsForDialect(d, "member_role_assignment"),
			plugins.ValidateColumnExistsForDialect(d, "member_role_assignment", "id"),
			plugins.ValidateColumnSpecForDialect(d, "member_role_assignment", "organization_id", core.ColumnSpec{Nullable: notNull}),
			plugins.ValidateColumnSpecForDialect(d, "member_role_assignment", "user_id", core.ColumnSpec{Nullable: notNull}),
			plugins.ValidateColumnSpecForDialect(d, "member_role_assignment", "role", core.ColumnSpec{Nullable: notNull}),
			// scope_type/scope_id must stay NOT NULL: the empty-string sentinel
			// for organization-wide assignments relies on the unique key
			// treating rows with the same scope as duplicates, which NULLs
			// would not.
			plugins.ValidateColumnSpecForDialect(d, "member_role_assignment", "scope_type", core.ColumnSpec{Nullable: notNull}),
			plugins.ValidateColumnSpecForDialect(d, "member_role_assignment", "scope_id", core.ColumnSpec{Nullable: notNull}),
			plugins.ValidateColumnExistsForDialect(d, "member_role_assignment", "created_at"),
			plugins.ValidateColumnExistsForDialect(d, "member_role_assignment", "updated_at"),
		}
	default:
		return []plugins.SchemaRequirement{}
	}
}

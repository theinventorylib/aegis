package oauth

import (
	_ "embed"
	"fmt"

	"github.com/theinventorylib/aegis/plugins"
)

// postgresAuthSchema contains auth.users prerequisite schema for PostgreSQL.
// OAuth requires the auth.users table to exist (foreign key constraint).
//
//go:embed internal/sql/postgres/auth_schema.sql
var postgresAuthSchema string

// postgresSchema contains the PostgreSQL-specific OAuth schema.
// Defines oauth_connections table with foreign key to auth.users.
//
//go:embed migrations/postgres/001_initial.up.sql
var postgresSchema string

// mysqlSchema contains the MySQL-specific OAuth schema.
//
//go:embed internal/sql/mysql/schema.sql
var mysqlSchema string

//go:embed internal/sql/sqlite/schema.sql
var sqliteSchema string

// GetSchema returns the initial database schema for the OAuth plugin.
//
// This function provides the CREATE TABLE statement for oauth_connections,
// formatted for the specified SQL dialect. For PostgreSQL, it includes both
// the auth schema prerequisite and the OAuth schema.
//
// Parameters:
//   - dialect: SQL dialect (DialectPostgres or DialectMySQL)
//
// Returns:
//   - *plugins.Schema: Schema metadata with SQL and version info
//   - error: Unsupported dialect error
func GetSchema(dialect plugins.Dialect) (*plugins.Schema, error) {
	var sql string

	switch dialect {
	case plugins.DialectPostgres:
		sql = postgresAuthSchema + "\n\n" + postgresSchema
	case plugins.DialectMySQL:
		sql = mysqlSchema
	case plugins.DialectSQLite:
		sql = sqliteSchema
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dialect)
	}

	return &plugins.Schema{
		SQL:     sql,
		Dialect: dialect,
		Info: plugins.SchemaInfo{
			Package:     "oauth",
			Version:     1,
			Description: "OAuth plugin schema",
		},
	}, nil
}

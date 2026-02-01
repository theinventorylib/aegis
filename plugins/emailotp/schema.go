package emailotp

import (
	_ "embed"
	"fmt"

	"github.com/theinventorylib/aegis/plugins"
)

//go:embed internal/sql/postgres/schema.sql
var postgresSchema string

//go:embed internal/sql/mysql/schema.sql
var mysqlSchema string

//go:embed internal/sql/sqlite/schema.sql
var sqliteSchema string

// GetSchema returns the database schema for the emailotp plugin.
//
// The schema extends the 'user' table with email-specific columns:
//   - email (VARCHAR, unique): User email address
//   - email_verified (BOOLEAN): Email verification status (default: false)
//
// These extensions enable email+password authentication and email verification.
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

	return &plugins.Schema{
		SQL:     sql,
		Dialect: dialect,
		Info: plugins.SchemaInfo{
			Package:     "emailotp",
			Version:     1,
			Description: "Email OTP plugin schema",
		},
	}, nil
}

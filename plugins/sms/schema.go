package sms

import (
	_ "embed"
	"fmt"

	"github.com/theinventorylib/aegis/plugins"
)

//go:embed internal/sql/postgres/schema.sql
var postgresSchema string

//go:embed internal/sql/mysql/schema.sql
var mysqlSchema string

// GetSchema returns the database schema for the SMS plugin.
//
// The schema extends the 'user' table with phone-specific columns:
//   - phone_number (VARCHAR, unique): User phone number in E.164 format
//   - phone_verified (BOOLEAN): Phone verification status (default: false)
//
// These extensions enable phone+password authentication and phone verification.
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
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dialect)
	}

	return &plugins.Schema{
		SQL:     sql,
		Dialect: dialect,
		Info: plugins.SchemaInfo{
			Package:     "sms",
			Version:     1,
			Description: "SMS plugin schema",
		},
	}, nil
}

package jwt

import (
	_ "embed"
	"fmt"

	"github.com/theinventorylib/aegis/plugins"
)

// postgresSchema contains the PostgreSQL-specific schema SQL.
// Embedded from: internal/sql/postgres/schema.sql
//
// Defines the jwks table with:
//   - kid (TEXT PRIMARY KEY): Unique key identifier
//   - key_data (TEXT NOT NULL): JSON-serialized JWK
//   - algorithm (TEXT NOT NULL): Signing algorithm (RS256, ES256, etc.)
//   - use (TEXT): Key usage (sig, enc)
//   - created_at (TIMESTAMP): Creation timestamp
//   - expires_at (TIMESTAMP): Optional expiration for key rotation
//
//go:embed internal/sql/postgres/schema.sql
var postgresSchema string

// mysqlSchema contains the MySQL-specific schema SQL.
// Embedded from: internal/sql/mysql/schema.sql
//
// Same structure as PostgreSQL but with MySQL-specific types:
//   - VARCHAR instead of TEXT for indexed columns
//   - DATETIME instead of TIMESTAMP
//
//go:embed internal/sql/mysql/schema.sql
var mysqlSchema string

// GetSchema returns the initial database schema for the JWT plugin.
//
// This function provides the CREATE TABLE statement for the jwks table,
// formatted for the specified SQL dialect. The schema is version 001
// (initial setup) and should be applied before any incremental migrations.
//
// Schema Purpose:
// The jwks table stores JSON Web Keys (JWKs) for JWT signing and verification.
// It supports key rotation by storing multiple keys with expiration timestamps.
//
// Supported Dialects:
//   - PostgreSQL: Uses TEXT and TIMESTAMP types
//   - MySQL: Uses VARCHAR and DATETIME types
//   - SQLite: Not currently supported (can use PostgreSQL schema)
//
// Parameters:
//   - dialect: SQL dialect (DialectPostgres or DialectMySQL)
//
// Returns:
//   - *plugins.Schema: Schema metadata with SQL and version info
//   - error: Unsupported dialect error
//
// Example:
//
//	schema, err := GetSchema(plugins.DialectPostgres)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("Schema version:", schema.Info.Version)
//	// Execute schema.SQL to create table
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

	info := plugins.SchemaInfo{
		Package:     "github.com/theinventorylib/aegis/plugins/jwt",
		Version:     1,
		Description: "JWT plugin schema",
	}

	return &plugins.Schema{
		Dialect: dialect,
		SQL:     sql,
		Info:    info,
	}, nil
}

// GetSchemaRequirements returns schema validation requirements for the JWT plugin.
//
// This function defines what database objects must exist for the plugin to function.
// The framework uses these requirements to validate the schema before plugin initialization.
//
// Requirements:
//   - Table "jwks" must exist: Stores JSON Web Keys for JWT signing/verification
//
// Validation Timing:
//   - Called during plugin.Init() after migrations are applied
//   - Plugin initialization fails if requirements are not met
//   - Prevents runtime errors from missing database objects
//
// Parameters:
//   - dialect: SQL dialect (DialectPostgres, DialectMySQL, etc.)
//
// Returns:
//   - []plugins.SchemaRequirement: List of validation checks to perform
//   - Empty slice if dialect is not supported
//
// Example:
//
//	reqs := GetSchemaRequirements(plugins.DialectPostgres)
//	for _, req := range reqs {
//	    if err := req.Validate(db); err != nil {
//	        log.Fatal("Schema validation failed:", err)
//	    }
//	}
func GetSchemaRequirements(dialect plugins.Dialect) []plugins.SchemaRequirement {
	switch dialect {
	case plugins.DialectPostgres, plugins.DialectMySQL:
		return []plugins.SchemaRequirement{
			// Table existence check for JWKS table
			plugins.ValidateTableExists("jwks"),
		}
	default:
		return []plugins.SchemaRequirement{}
	}
}

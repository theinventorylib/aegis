// Package auth provides schema export functionality for different database dialects.
// The actual SQL schemas are embedded from internal files and can be accessed
// programmatically for documentation, CLI tools, or custom migration systems.
//
// This allows users to access the core authentication schema definitions without
// needing to extract them from binary builds.
package auth

import (
	_ "embed"
	"fmt"

	authtypes "github.com/theinventorylib/aegis/auth/types"
)

// Embedded schema SQL for each supported database dialect.
// These are the initial schema definitions (migration version 001).

//go:embed migrations/postgres/001_initial.up.sql
var postgresSchema string

//go:embed migrations/mysql/001_initial.up.sql
var mysqlSchema string

//go:embed migrations/sqlite/001_initial.up.sql
var sqliteSchema string

// Dialect represents a supported database engine.
type Dialect = authtypes.Dialect

const (
	// DialectPostgres is for PostgreSQL databases (>=9.6 recommended)
	DialectPostgres = authtypes.DialectPostgres

	// DialectMySQL is for MySQL databases (>=5.7 or MariaDB >=10.2)
	DialectMySQL = authtypes.DialectMySQL

	// DialectSQLite is for SQLite databases
	DialectSQLite = authtypes.DialectSQLite
)

// SchemaInfo contains metadata about a schema package.
// This can be used for dependency tracking and versioning in complex systems.
type SchemaInfo struct {
	// Package is the Go import path for this schema
	Package string

	// Version is the schema version number (currently unused, always 0)
	Version int

	// Description is a human-readable summary of the schema
	Description string

	// Dependencies lists other schema packages this schema depends on
	Dependencies []Dependency
}

// Dependency represents a schema dependency on another package.
type Dependency struct {
	// Package is the Go import path of the dependency
	Package string

	// Version is the minimum required version of the dependency
	Version int
}

// Schema represents the complete SQL schema for a specific database dialect.
type Schema struct {
	// Dialect identifies the database type (postgres, mysql, sqlite)
	Dialect Dialect

	// SQL is the complete schema definition in SQL
	SQL string

	// Info contains metadata about the schema package
	Info SchemaInfo
}

// GetSchema returns the complete SQL schema definition for the specified dialect.
//
// This is useful for:
//   - Generating documentation
//   - Initializing new databases
//   - Comparing schemas across dialects
//   - Custom migration tooling
//
// The returned Schema includes both the raw SQL and metadata about the schema package.
//
// Example:
//
//	schema, err := auth.GetSchema(auth.DialectPostgres)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(schema.SQL) // Prints the full PostgreSQL schema
func GetSchema(dialect Dialect) (*Schema, error) {
	var sql string

	switch dialect {
	case DialectPostgres:
		sql = postgresSchema
	case DialectMySQL:
		sql = mysqlSchema
	case DialectSQLite:
		sql = sqliteSchema
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dialect)
	}

	info := parseSchemaInfo(sql)

	return &Schema{
		Dialect: dialect,
		SQL:     sql,
		Info:    info,
	}, nil
}

// parseSchemaInfo returns default metadata for the schema.
// Currently this returns static metadata. In the future, this could parse
// metadata comments from the SQL file itself.
func parseSchemaInfo(_ string) SchemaInfo {
	// Simple parser/regex could go here, for now returns default
	return SchemaInfo{
		Package:      "github.com/theinventorylib/aegis/auth",
		Version:      0, // Version tracking not yet implemented
		Description:  "",
		Dependencies: []Dependency{},
	}
}

// PackageName returns the package identifier for the auth schema.
// This is used by the CLI schema export tool.
func PackageName() string {
	return "auth"
}

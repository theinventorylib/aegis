// Package types defines the core domain models, interfaces, and types for the Aegis authentication module.
package types

// Dialect represents a supported database engine.
type Dialect string

const (
	// DialectPostgres is for PostgreSQL databases (>=9.6 recommended)
	DialectPostgres Dialect = "postgres"

	// DialectMySQL is for MySQL databases (>=5.7 or MariaDB >=10.2)
	DialectMySQL Dialect = "mysql"

	// DialectSQLite is for SQLite databases
	DialectSQLite Dialect = "sqlite"
)

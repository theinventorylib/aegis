package core

import (
	"fmt"
	"strings"
)

// Schema dialect constants used by the dialect-aware ValidateXxxForDialect
// helpers. These mirror config.Dialect values without importing the config
// package (which would create an import cycle, since config imports core).
const (
	SchemaDialectPostgres = "postgres"
	SchemaDialectMySQL    = "mysql"
	SchemaDialectSQLite   = "sqlite"
)

// ValidateTableExistsForDialect builds a table-existence requirement using
// the appropriate metadata source for the given dialect.
//
// PostgreSQL and MySQL use information_schema. SQLite has no
// information_schema; instead it exposes sqlite_master, which we query
// directly. Unknown dialects fall back to information_schema for
// backwards compatibility with the original ValidateTableExists helper.
func ValidateTableExistsForDialect(dialect, tableName string) SchemaRequirement {
	switch strings.ToLower(dialect) {
	case SchemaDialectSQLite:
		tableName = SanitizeSQLIdentifier(tableName)
		return SchemaRequirement{
			Name:        fmt.Sprintf("Table '%s' exists", tableName),
			Table:       tableName,
			Query:       fmt.Sprintf("SELECT name FROM sqlite_master WHERE type='table' AND name='%s' LIMIT 1", tableName),
			Description: fmt.Sprintf("Table '%s' does not exist", tableName),
		}
	default:
		return ValidateTableExists(tableName)
	}
}

// ValidateColumnExistsForDialect builds a column-existence requirement
// using the appropriate metadata source for the given dialect.
//
// SQLite returns one row per column from pragma_table_info(<table>); we
// filter by column name. Other dialects use information_schema.
func ValidateColumnExistsForDialect(dialect, tableName, columnName string) SchemaRequirement {
	switch strings.ToLower(dialect) {
	case SchemaDialectSQLite:
		tableName = SanitizeSQLIdentifier(tableName)
		columnName = SanitizeSQLIdentifier(columnName)
		return SchemaRequirement{
			Name:  fmt.Sprintf("Column '%s.%s' exists", tableName, columnName),
			Table: tableName,
			Query: fmt.Sprintf(
				"SELECT name FROM pragma_table_info('%s') WHERE name='%s' LIMIT 1",
				tableName, columnName,
			),
			Description: fmt.Sprintf("Column '%s' does not exist in table '%s'", columnName, tableName),
		}
	default:
		return ValidateColumnExists(tableName, columnName)
	}
}

// ValidateColumnSpecForDialect builds a column-spec requirement using the
// appropriate metadata source for the given dialect.
//
// SQLite's pragma_table_info exposes `type` (text) and `notnull` (0/1)
// instead of the information_schema column names. We translate the spec
// fields onto those columns. Type comparisons remain
// case-insensitive-equal; SQLite stores types verbatim from the CREATE
// TABLE declaration, so callers should pass the exact declared type
// (e.g. "TEXT", "INTEGER", "BLOB").
func ValidateColumnSpecForDialect(dialect, tableName, columnName string, spec ColumnSpec) SchemaRequirement {
	if strings.ToLower(dialect) != SchemaDialectSQLite {
		return ValidateColumnSpec(tableName, columnName, spec)
	}

	tableName = SanitizeSQLIdentifier(tableName)
	columnName = SanitizeSQLIdentifier(columnName)

	conditions := []string{fmt.Sprintf("name='%s'", columnName)}
	descParts := []string{fmt.Sprintf("column '%s.%s'", tableName, columnName)}

	if spec.DataType != "" {
		dt := sanitizeSQLLiteral(spec.DataType)
		conditions = append(conditions, fmt.Sprintf("LOWER(type) = LOWER('%s')", dt))
		descParts = append(descParts, fmt.Sprintf("data_type=%s", spec.DataType))
	}
	if spec.Nullable != nil {
		// pragma_table_info.notnull is 1 when NOT NULL is declared, 0 otherwise.
		want := "0"
		if !*spec.Nullable {
			want = "1"
		}
		conditions = append(conditions, fmt.Sprintf("notnull = %s", want))
		descParts = append(descParts, fmt.Sprintf("nullable=%v", *spec.Nullable))
	}

	return SchemaRequirement{
		Name:  fmt.Sprintf("Column spec '%s.%s' matches", tableName, columnName),
		Table: tableName,
		Query: fmt.Sprintf(
			"SELECT name FROM pragma_table_info('%s') WHERE %s LIMIT 1",
			tableName, joinAnd(conditions),
		),
		Description: fmt.Sprintf("expected %s — column missing or schema drift detected", joinComma(descParts)),
	}
}
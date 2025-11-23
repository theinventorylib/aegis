package migrations

import (
	_ "embed"
)

//go:embed schema.sql
var CoreSchemaSQL string

// CoreMigration represents the core Aegis schema migration
var CoreMigration = MigrationInfo{
	Namespace:   "core",
	Version:     "001",
	Description: "Aegis core schema - user, accounts, verification, session, jwks",
	Up:          CoreSchemaSQL,
	Down: `
DROP TABLE IF EXISTS auth.jwks;
DROP TABLE IF EXISTS auth.session CASCADE;
DROP TABLE IF EXISTS auth.verification;
DROP TABLE IF EXISTS auth.accounts CASCADE;
DROP TABLE IF EXISTS auth.user CASCADE;
DROP FUNCTION IF EXISTS auth.update_updated_at_column();
`,
}

// MigrationInfo represents metadata about a migration
type MigrationInfo struct {
	Namespace   string // "core" or plugin name
	Version     string
	Description string
	Up          string
	Down        string
}

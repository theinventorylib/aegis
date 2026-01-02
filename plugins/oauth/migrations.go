// Package oauth provides database migration management for the OAuth plugin.
//
// This file handles parsing and loading migration files from the embedded filesystem,
// supporting multiple SQL dialects (PostgreSQL, MySQL, SQLite).
//
// Migration Versioning:
//   - Version 001: Initial schema (oauth_connections table)
//   - Version 002+: Additional migrations (indexes, columns, etc.)
//
// File Naming Convention:
//   - Up migrations: 002_add_index.up.sql
//   - Down migrations: 002_add_index.down.sql
package oauth

import (
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/theinventorylib/aegis/plugins"
)

// schemaFS embeds initial schema files for each SQL dialect.
//
//go:embed internal/sql/*/*.sql
var schemaFS embed.FS

// migrationFS embeds incremental migration files for each SQL dialect.
//
//go:embed migrations/*/*.sql
var migrationFS embed.FS

// GetMigrations returns all database migrations for the specified SQL dialect.
//
// This function combines the initial schema (version 001) with any additional
// migrations (version 002+) to produce a complete, ordered list of migrations.
//
// Parameters:
//   - dialect: SQL dialect (DialectPostgres, DialectMySQL, DialectSQLite)
//
// Returns:
//   - []plugins.Migration: Ordered list of migrations (version ASC)
//   - error: File read error or unsupported dialect
func GetMigrations(dialect plugins.Dialect) ([]plugins.Migration, error) {
	// Load initial schema as version 001
	schemaPath := fmt.Sprintf("internal/sql/%s/schema.sql", dialect)
	schemaContent, err := schemaFS.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("read schema file for %s: %w", dialect, err)
	}
	initial := plugins.Migration{
		Version:     1,
		Description: "initial",
		Up:          string(schemaContent),
		Down:        "", // No down for initial schema
	}

	migrations := make(map[int]*plugins.Migration)
	migrations[1] = &initial

	// Load additional migrations
	dir := fmt.Sprintf("migrations/%s", dialect)
	entries, err := migrationFS.ReadDir(dir)
	if err != nil {
		// If no migrations dir, just return initial
		if strings.Contains(err.Error(), "no such file") {
			return []plugins.Migration{initial}, nil
		}
		return nil, fmt.Errorf("read migrations dir for %s: %w", dialect, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		// Parse name: version_description.type.sql
		parts := strings.SplitN(name, "_", 2)
		if len(parts) != 2 {
			continue
		}
		versionStr := parts[0]
		rest := parts[1]

		// rest: description.type.sql
		descParts := strings.SplitN(rest, ".", 2)
		if len(descParts) != 2 {
			continue
		}
		description := descParts[0]
		typeExt := descParts[1] // type.sql

		if !strings.HasSuffix(typeExt, ".sql") {
			continue
		}
		migType := strings.TrimSuffix(typeExt, ".sql") // up or down

		version, err := strconv.Atoi(versionStr)
		if err != nil || version < 2 { // Skip version 001 as it's handled by schema
			continue
		}

		content, err := migrationFS.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("read migration file %s: %w", name, err)
		}
		sql := string(content)

		if migrations[version] == nil {
			migrations[version] = &plugins.Migration{Version: version, Description: description}
		}
		switch migType {
		case "up":
			migrations[version].Up = sql
		case "down":
			migrations[version].Down = sql
		}
	}

	// Convert map to slice
	result := make([]plugins.Migration, 0, len(migrations))
	for _, mig := range migrations {
		result = append(result, *mig)
	}

	// Sort by version
	sort.Slice(result, func(i, j int) bool {
		return result[i].Version < result[j].Version
	})

	return result, nil
}

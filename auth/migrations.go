package auth

import (
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// migrationFS embeds all migration files for schema evolution.
// These represent all migrations (version 001+).
//
//go:embed migrations/*/*.sql
var migrationFS embed.FS

// Migration represents a versioned database schema change.
// Each migration has both an "up" script (to apply the change) and a "down"
// script (to revert it), enabling bidirectional schema evolution.
type Migration struct {
	// Version is the numeric migration version (e.g., 1, 2, 3).
	Version int

	// Description is a human-readable summary of what this migration does
	// (e.g., "initial", "add_user_roles", "alter_session_index").
	Description string

	// Up is the SQL to apply this migration (create tables, add columns, etc.).
	Up string

	// Down is the SQL to revert this migration (drop tables, remove columns, etc.).
	Down string
}

// GetMigrations returns all migrations for the specified database dialect in version order.
//
// Migration versioning:
//   - Version 001+: All migrations from migrations/<dialect>/*.sql
//
// Migration file naming convention:
//
//	<version>_<description>.<up|down>.sql
//
// Examples:
//
//	001_initial.up.sql     - Initial schema migration
//	001_initial.down.sql   - Revert initial schema
//	002_add_user_roles.up.sql     - Applies migration 002
//	002_add_user_roles.down.sql   - Reverts migration 002
//	003_alter_sessions.up.sql     - Applies migration 003
//
// Each migration version must have both .up.sql and .down.sql files.
// Migrations are returned sorted by version number for sequential application.
//
// Parameters:
//   - dialect: The database dialect (postgres, mysql, sqlite)
//
// Returns:
//   - Slice of migrations sorted by version
//   - Error if the dialect is not supported or if migration files are malformed
func GetMigrations(dialect Dialect) ([]Migration, error) {
	migrations := make(map[int]*Migration)

	// Load all migrations
	dir := fmt.Sprintf("migrations/%s", dialect)
	entries, err := migrationFS.ReadDir(dir)
	if err != nil {
		// If no migrations dir, return empty
		if strings.Contains(err.Error(), "file does not exist") {
			return []Migration{}, nil
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

		// Parse migration filename: "<version>_<description>.<type>.sql"
		// Example: "001_initial.up.sql" -> version=001, description="initial", type="up"
		parts := strings.SplitN(name, "_", 2)
		if len(parts) != 2 {
			continue // Skip malformed filenames
		}
		versionStr := parts[0]
		rest := parts[1] // "description.type.sql"

		// Split description and type: "initial.up.sql" -> "initial", "up.sql"
		descParts := strings.SplitN(rest, ".", 2)
		if len(descParts) != 2 {
			continue // Missing extension
		}
		description := descParts[0]
		typeExt := descParts[1] // "up.sql" or "down.sql"

		if !strings.HasSuffix(typeExt, ".sql") {
			continue
		}
		migType := strings.TrimSuffix(typeExt, ".sql") // "up" or "down"

		// Parse version number (must be >= 1)
		version, err := strconv.Atoi(versionStr)
		if err != nil || version < 1 {
			continue // Skip invalid versions
		}

		content, err := migrationFS.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("read migration file %s: %w", name, err)
		}
		sql := string(content)

		// Accumulate up/down SQL for each version. Multiple files with the same
		// version number (one .up.sql and one .down.sql) are merged into a single
		// Migration struct.
		if migrations[version] == nil {
			migrations[version] = &Migration{Version: version, Description: description}
		}
		switch migType {
		case "up":
			migrations[version].Up = sql
		case "down":
			migrations[version].Down = sql
		}
	}

	// Convert map to slice
	result := make([]Migration, 0, len(migrations))
	for _, mig := range migrations {
		result = append(result, *mig)
	}

	// Sort by version
	sort.Slice(result, func(i, j int) bool {
		return result[i].Version < result[j].Version
	})

	return result, nil
}

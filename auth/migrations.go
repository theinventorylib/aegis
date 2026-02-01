package auth

import (
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// schemaFS embeds the initial SQL schema files for different database dialects.
// These files contain the baseline table definitions (version 001).
//
//go:embed internal/sql/*/*.sql
var schemaFS embed.FS

// migrationFS embeds incremental migration files for schema evolution.
// These represent changes made after the initial schema (version 002+).
//
//go:embed migrations/*/*.sql
var migrationFS embed.FS

// Migration represents a versioned database schema change.
// Each migration has both an "up" script (to apply the change) and a "down"
// script (to revert it), enabling bidirectional schema evolution.
type Migration struct {
	// Version is the numeric migration version (e.g., 1, 2, 3).
	// Version 1 is always the initial schema.
	Version int

	// Description is a human-readable summary of what this migration does
	// (e.g., "add_user_roles", "alter_session_index").
	Description string

	// Up is the SQL to apply this migration (create tables, add columns, etc.).
	Up string

	// Down is the SQL to revert this migration (drop tables, remove columns, etc.).
	Down string
}

// GetMigrations returns all migrations for the specified database dialect in version order.
//
// Migration versioning:
//   - Version 001: The initial schema from internal/sql/<dialect>/schema.sql
//   - Version 002+: Additional migrations from migrations/<dialect>/*.sql
//
// Migration file naming convention:
//
//	<version>_<description>.<up|down>.sql
//
// Examples:
//
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
	// Load initial schema as version 001
	schemaPath := fmt.Sprintf("internal/sql/%s/schema.sql", dialect)
	schemaContent, err := schemaFS.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("read schema file for %s: %w", dialect, err)
	}
	initial := Migration{
		Version:     1,
		Description: "initial",
		Up:          string(schemaContent),
		Down: `
-- Core schema cleanup
DROP TABLE IF EXISTS session;
DROP TABLE IF EXISTS verification;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS user;
`,
	}

	migrations := make(map[int]*Migration)
	migrations[1] = &initial

	// Load additional migrations
	dir := fmt.Sprintf("migrations/%s", dialect)
	entries, err := migrationFS.ReadDir(dir)
	if err != nil {
		// If no migrations dir, just return initial
		if strings.Contains(err.Error(), "file does not exist") {
			return []Migration{initial}, nil
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
		// Example: "002_add_roles.up.sql" -> version=002, description="add_roles", type="up"
		parts := strings.SplitN(name, "_", 2)
		if len(parts) != 2 {
			continue // Skip malformed filenames
		}
		versionStr := parts[0]
		rest := parts[1] // "description.type.sql"

		// Split description and type: "add_roles.up.sql" -> "add_roles", "up.sql"
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

		// Parse version number (must be >= 2 since version 1 is the base schema)
		version, err := strconv.Atoi(versionStr)
		if err != nil || version < 2 {
			continue // Skip version 001 (handled by schema.sql) and invalid versions
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

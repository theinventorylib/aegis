package organizations

import (
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/theinventorylib/aegis/plugins"
)

//go:embed internal/sql/*/*.sql
var schemaFS embed.FS

//go:embed migrations/*/*.sql
var migrationFS embed.FS

// GetMigrations returns all database migrations for the organizations plugin.
//
// This function loads migrations from embedded SQL files and returns them in
// version order. The initial schema is always treated as version 001.
//
// Version Numbering:
//   - Version 001: Initial schema from internal/sql/<dialect>/schema.sql
//   - Version 002+: Additional migrations from migrations/<dialect>/<version>_<description>.<up|down>.sql
//
// Migration File Format:
//   - Up migration: 002_add_teams.up.sql
//   - Down migration: 002_add_teams.down.sql
//
// Parameters:
//   - dialect: Database dialect (postgres, mysql, sqlite)
//
// Returns:
//   - []plugins.Migration: Sorted list of migrations (oldest first)
//   - error: If schema files cannot be read or parsed
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
		Down:        "", // No down migration for initial schema
	}

	migrations := make(map[int]*plugins.Migration)
	migrations[1] = &initial

	// Load additional migrations from migrations/<dialect>/ directory
	dir := fmt.Sprintf("migrations/%s", dialect)
	entries, err := migrationFS.ReadDir(dir)
	if err != nil {
		// If no migrations directory exists, return only initial schema
		if strings.Contains(err.Error(), "no such file") {
			return []plugins.Migration{initial}, nil
		}
		return nil, fmt.Errorf("read migrations dir for %s: %w", dialect, err)
	}

	// Parse migration files and build version map
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		// Parse filename format: <version>_<description>.<up|down>.sql
		// Example: 002_add_teams.up.sql
		parts := strings.SplitN(name, "_", 2)
		if len(parts) != 2 {
			continue
		}
		versionStr := parts[0]
		rest := parts[1]

		// Parse description and migration type
		// rest format: description.type.sql
		descParts := strings.SplitN(rest, ".", 2)
		if len(descParts) != 2 {
			continue
		}
		description := descParts[0]
		typeExt := descParts[1] // type.sql

		if !strings.HasSuffix(typeExt, ".sql") {
			continue
		}
		migType := strings.TrimSuffix(typeExt, ".sql") // "up" or "down"

		// Parse version number (must be >= 002)
		version, err := strconv.Atoi(versionStr)
		if err != nil || version < 2 { // Skip version 001 (handled by schema)
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

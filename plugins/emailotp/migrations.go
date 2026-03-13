package emailotp

import (
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/theinventorylib/aegis/plugins"
)

//go:embed migrations/*/*.sql
var migrationFS embed.FS

// GetMigrations returns all database migrations for the emailotp plugin.
//
// This function loads migrations from embedded SQL files and returns them in
// version order.
//
// Version Numbering:
//   - Version 001+: Migrations from migrations/<dialect>/<version>_<description>.<up|down>.sql
//
// Migration File Format:
//   - Up migration: 001_initial.up.sql
//   - Down migration: 001_initial.down.sql
//
// Parameters:
//   - dialect: Database dialect (postgres, mysql, sqlite)
//
// Returns:
//   - []plugins.Migration: Sorted list of migrations (oldest first)
//   - error: If migration files cannot be read or parsed
func GetMigrations(dialect plugins.Dialect) ([]plugins.Migration, error) {
	migrations := make(map[int]*plugins.Migration)

	// Load all migrations
	dir := fmt.Sprintf("migrations/%s", dialect)
	entries, err := migrationFS.ReadDir(dir)
	if err != nil {
		// If no migrations dir, return empty
		if strings.Contains(err.Error(), "file does not exist") {
			return []plugins.Migration{}, nil
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
		if err != nil || version < 1 {
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

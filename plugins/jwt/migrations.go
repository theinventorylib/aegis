// Package jwt provides database migration management for the JWT plugin.
//
// This file handles parsing and loading migration files from the embedded filesystem,
// supporting multiple SQL dialects (PostgreSQL, MySQL, SQLite).
//
// Migration Versioning:
//   - Version 001+: Migrations from migrations/<dialect>/<version>_<description>.<up|down>.sql
//
// File Naming Convention:
//   - Up migrations: 001_initial.up.sql
//   - Down migrations: 001_initial.down.sql
//
// Directory Structure:
//
//	jwt/
//	  migrations/
//	    postgres/
//	      002_altered.up.sql
//	      002_altered.down.sql
//	    mysql/
//	      002_altered.up.sql
//	      002_altered.down.sql
//
// The migration system ensures the correct schema version is applied for each
// SQL dialect, supporting forward (up) and backward (down) migrations.
package jwt

import (
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/theinventorylib/aegis/plugins"
)

// migrationFS embeds incremental migration files for each SQL dialect.
// Files are located at: migrations/<dialect>/<version>_<description>.<up|down>.sql
//
//go:embed migrations/*/*.sql
var migrationFS embed.FS

// GetMigrations returns all database migrations for the specified SQL dialect.
//
// This function gets all migrations to produce a complete, ordered list of migrations.
//
// Version Numbering:
//   - Version 001+: Additional migrations from migrations/<dialect>/ directory
//
// File Naming:
//   - Format: <version>_<description>.<type>.sql
//   - Example: 001_initial.up.sql
//   - Type: "up" (apply) or "down" (rollback)
//
// Migration Loading:
//  1. Scan migrations/<dialect>/ for files
//  2. Parse version number and type (up/down) from filename
//  3. Group up/down migrations by version
//  4. Sort by version number ascending
//  5. Return ordered migration list
//
// Parameters:
//   - dialect: SQL dialect (DialectPostgres, DialectMySQL, DialectSQLite)
//
// Returns:
//   - []plugins.Migration: Ordered list of migrations (version ASC)
//   - error: File read error, invalid filename, or unsupported dialect
//
// Example:
//
//	migrations, err := GetMigrations(plugins.DialectPostgres)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, m := range migrations {
//	    fmt.Printf("Version %d: %s\n", m.Version, m.Description)
//	    // Execute m.Up SQL to apply migration
//	}
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

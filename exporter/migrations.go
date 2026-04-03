// Package exporter provides functionality for exporting database migrations and schemas.
package exporter

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/theinventorylib/aegis/auth"
	"github.com/theinventorylib/aegis/plugins"
)

// ExportFormat defines the output format for migration exports.
type ExportFormat string

const (
	// FormatSQL exports migrations as plain SQL files.
	FormatSQL ExportFormat = "sql"
	// FormatGoose exports migrations in Goose format.
	FormatGoose ExportFormat = "goose"
	// FormatGolangMigrate exports migrations in golang-migrate format.
	FormatGolangMigrate ExportFormat = "golang-migrate"
)

// Migration represents a single migration.
type Migration struct {
	Number      int
	Up          string
	Down        string
	Version     string
	Description string
}

// Config configures the migration exporter.
type Config struct {
	Format    ExportFormat
	Dialect   plugins.Dialect
	OutputDir string
	AuthOnly  bool
	Plugins   []plugins.Plugin // Plugin instances to export
}

// Exporter handles exporting migrations to various formats.
type Exporter struct {
	format    ExportFormat
	outputDir string
	dialect   plugins.Dialect
	authOnly  bool
	plugins   []plugins.Plugin
}

// NewExporter creates a new migration exporter with the provided configuration.
func NewExporter(config Config) *Exporter {
	return &Exporter{
		format:    config.Format,
		outputDir: config.OutputDir,
		dialect:   config.Dialect,
		authOnly:  config.AuthOnly,
		plugins:   config.Plugins,
	}
}

// numberedMigration pairs a globally-assigned sequence number with a migration
// and the name of the source that produced it (e.g. "auth", "jwt").
type numberedMigration struct {
	GlobalNum int
	Source    string
	Migration
}

// collectAllMigrations assembles every migration that should be written and
// assigns each a globally sequential number.
//
// Ordering:
//  1. Auth migrations, sorted by their internal version number.
//  2. Plugin migrations, plugins sorted alphabetically by name, each plugin's
//     migrations sorted by their internal version number.
//
// Because the output directory is always cleaned before writing, there is no
// need to reserve numeric blocks or check for offset collisions — numbers are
// simply assigned 1, 2, 3 … in order.
func (e *Exporter) collectAllMigrations() ([]numberedMigration, error) {
	var result []numberedMigration
	counter := 1

	authMigrations, err := e.getAuthMigrations()
	if err != nil {
		return nil, err
	}
	for _, m := range authMigrations {
		result = append(result, numberedMigration{GlobalNum: counter, Source: "auth", Migration: m})
		counter++
	}

	if !e.authOnly {
		// Sort plugins alphabetically so the output order is deterministic
		// regardless of the order they were passed to the exporter.
		pluginList := make([]plugins.Plugin, len(e.plugins))
		copy(pluginList, e.plugins)
		sort.Slice(pluginList, func(i, j int) bool {
			return pluginList[i].Name() < pluginList[j].Name()
		})
		for _, p := range pluginList {
			for _, m := range e.getPluginMigrations(p) {
				result = append(result, numberedMigration{GlobalNum: counter, Source: p.Name(), Migration: m})
				counter++
			}
		}
	}

	return result, nil
}

// Export writes migrations to disk in the specified format.
//
// Files are placed directly in OutputDir with sequentially-assigned numeric
// prefixes: auth migrations first, then plugin migrations in alphabetical
// plugin-name order.
//
// Any file whose name contains "_aegis_" that already exists in OutputDir is
// removed before new files are written. This keeps the folder in sync with the
// current Aegis version without disturbing files the developer created
// themselves (which must not use "_aegis_" in their own filenames).
func (e *Exporter) Export() error {
	if err := os.MkdirAll(e.outputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := e.cleanAegisFiles(); err != nil {
		return fmt.Errorf("failed to clean existing aegis files: %w", err)
	}

	var exportErr error
	switch e.format {
	case FormatSQL:
		exportErr = e.exportSQL()
	case FormatGoose:
		exportErr = e.exportGoose()
	case FormatGolangMigrate:
		exportErr = e.exportGolangMigrate()
	default:
		return fmt.Errorf("unsupported format: %s", e.format)
	}

	// Generate README
	if err := e.generateReadme(); err != nil {
		return fmt.Errorf("generate readme: %w", err)
	}

	return exportErr
}

// cleanAegisFiles removes every file in OutputDir whose name contains "_aegis_".
// This naming pattern is reserved for Aegis-generated migration files, so the
// developer's own files (which must not use that substring) are left untouched.
// Running this before writing ensures that renamed or removed aegis migrations
// do not linger inside the user's migrations folder.
func (e *Exporter) cleanAegisFiles() error {
	entries, err := os.ReadDir(e.outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // directory not created yet — nothing to clean
		}
		return fmt.Errorf("reading output directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.Contains(entry.Name(), "_aegis_") {
			path := filepath.Join(e.outputDir, entry.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("removing stale aegis file %q: %w", entry.Name(), err)
			}
		}
	}
	return nil
}

func (e *Exporter) getAuthMigrations() ([]Migration, error) {
	allMigrations, err := auth.GetMigrations(e.dialect)
	if err != nil {
		return nil, fmt.Errorf("get auth migrations: %w", err)
	}

	migrations := make([]Migration, 0, len(allMigrations))
	for _, am := range allMigrations {
		migrations = append(migrations, Migration{
			Number:      am.Version,
			Up:          am.Up,
			Down:        am.Down,
			Version:     am.Description,
			Description: am.Description,
		})
	}
	return migrations, nil
}

func (e *Exporter) getPluginMigrations(plugin plugins.Plugin) []Migration {
	pluginMigrations := plugin.GetMigrations()

	migrations := make([]Migration, 0, len(pluginMigrations))
	for _, pm := range pluginMigrations {
		migrations = append(migrations, Migration{
			Number:      pm.Version,
			Up:          pm.Up,
			Down:        pm.Down,
			Version:     pm.Description,
			Description: pm.Description,
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Number < migrations[j].Number
	})

	return migrations
}

// exportSQL exports migrations as plain SQL files into a flat directory.
func (e *Exporter) exportSQL() error {
	all, err := e.collectAllMigrations()
	if err != nil {
		return err
	}
	for _, nm := range all {
		sourceName := strings.ReplaceAll(nm.Source, "-", "_")
		name := fmt.Sprintf("%05d_aegis_%s_%s.sql", nm.GlobalNum, sourceName, nm.Version)
		content := fmt.Sprintf("-- Aegis: %s\n-- Version: %s\n-- Description: %s\n\n%s",
			nm.Source, nm.Version, nm.Description, nm.Up)
		if err := os.WriteFile(filepath.Join(e.outputDir, name), []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to write %s migration: %w", nm.Source, err)
		}
	}
	return nil
}

// exportGoose exports migrations in Goose format into a flat directory.
func (e *Exporter) exportGoose() error {
	all, err := e.collectAllMigrations()
	if err != nil {
		return err
	}
	for _, nm := range all {
		sourceName := strings.ReplaceAll(nm.Source, "-", "_")
		name := fmt.Sprintf("%05d_aegis_%s_%s.sql", nm.GlobalNum, sourceName, nm.Version)
		content := fmt.Sprintf(
			"-- +goose Up\n-- +goose StatementBegin\n%s\n-- +goose StatementEnd\n\n-- +goose Down\n-- +goose StatementBegin\n%s\n-- +goose StatementEnd\n",
			nm.Up, nm.Down)
		if err := os.WriteFile(filepath.Join(e.outputDir, name), []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to write %s migration: %w", nm.Source, err)
		}
	}
	return nil
}

// exportGolangMigrate exports migrations in golang-migrate format into a flat
// directory. golang-migrate uses 6-digit numeric prefixes and paired
// .up.sql / .down.sql files.
func (e *Exporter) exportGolangMigrate() error {
	all, err := e.collectAllMigrations()
	if err != nil {
		return err
	}
	for _, nm := range all {
		sourceName := strings.ReplaceAll(nm.Source, "-", "_")
		base := filepath.Join(e.outputDir, fmt.Sprintf("%06d_aegis_%s_%s", nm.GlobalNum, sourceName, nm.Version))
		if err := os.WriteFile(base+".up.sql", []byte(nm.Up), 0600); err != nil {
			return fmt.Errorf("failed to write %s up migration: %w", nm.Source, err)
		}
		if err := os.WriteFile(base+".down.sql", []byte(nm.Down), 0600); err != nil {
			return fmt.Errorf("failed to write %s down migration: %w", nm.Source, err)
		}
	}
	return nil
}

func (e *Exporter) generateReadme() error {
	pluginNames := make([]string, 0, len(e.plugins))
	for _, p := range e.plugins {
		pluginNames = append(pluginNames, p.Name())
	}
	sort.Strings(pluginNames)

	content := fmt.Sprintf(`# Aegis Migrations

These migrations were exported from Aegis authentication library.

## Configuration

- **Dialect:** %s
- **Format:** %s
- **Auth Only:** %v
- **Included Plugins:** %s

## Numbering Scheme

Migrations are numbered sequentially starting at 1: auth migrations first,
then plugin migrations in alphabetical plugin-name order. Re-running
"aegis export" replaces all Aegis-generated files and reassigns numbers from
scratch, so the folder is always consistent with your current Aegis version.

## Running Migrations

### Using Goose

`+"```bash"+`
goose -dir %s %s up
`+"```"+`

### Using golang-migrate

`+"```bash"+`
migrate -path %s -database "%s://localhost/mydb" up
`+"```"+`

## Plugin Information

`,
		e.dialect,
		e.format,
		e.authOnly,
		strings.Join(pluginNames, ", "),
		e.outputDir, e.dialect,
		e.outputDir, e.dialect,
	)

	if len(e.plugins) > 0 {
		content += "| Plugin | Version | Description |\n"
		content += "|--------|---------|-------------|\n"
		for _, p := range e.plugins {
			content += fmt.Sprintf("| %s | %s | %s |\n", p.Name(), p.Version(), p.Description())
		}
	} else {
		content += "No plugins included in this export.\n"
	}

	content += `
## Migration Sources

The original migrations are available at:
- Auth: github.com/theinventorylib/aegis/auth/migrations/[dialect]/
- Plugins: github.com/theinventorylib/aegis/plugins/[name]/migrations/[dialect]/
`

	return os.WriteFile(filepath.Join(e.outputDir, "README.md"), []byte(content), 0600)
}

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

// Migration represents a single migration
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

// Exporter handles exporting migrations to various formats
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

// Export writes migrations to disk in the specified format
func (e *Exporter) Export() error {
	if err := os.MkdirAll(e.outputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
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

	if exportErr != nil {
		return exportErr
	}

	return nil
}

func (e *Exporter) getAuthMigrations() ([]Migration, error) {
	// Map plugins.Dialect to auth.Dialect
	var authDialect auth.Dialect
	switch e.dialect {
	case plugins.DialectPostgres:
		authDialect = auth.DialectPostgres
	case plugins.DialectMySQL:
		authDialect = auth.DialectMySQL
	case plugins.DialectSQLite:
		authDialect = auth.DialectSQLite
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", e.dialect)
	}

	// Get migrations from auth package
	allMigrations, err := auth.GetMigrations(authDialect)
	if err != nil {
		return nil, fmt.Errorf("get auth migrations: %w", err)
	}

	// Convert to exporter Migration format
	migrations := make([]Migration, 0, len(allMigrations))
	for _, am := range allMigrations {
		migrations = append(migrations, Migration{
			Number:      am.Version,
			Up:          am.Up,
			Down:        am.Down,
			Version:     fmt.Sprintf("%d", am.Version),
			Description: am.Description,
		})
	}

	return migrations, nil
}

func (e *Exporter) getPluginMigrations(plugin plugins.Plugin) []Migration {
	// Get migrations directly from the plugin instance
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

// exportSQL exports migrations as plain SQL files
func (e *Exporter) exportSQL() error {
	authMigrations, err := e.getAuthMigrations()
	if err != nil {
		return err
	}

	migrationCounter := 1
	for _, migration := range authMigrations {
		filename := filepath.Join(e.outputDir, fmt.Sprintf("%03d_aegis_auth_%s.sql", migrationCounter, migration.Version))
		content := fmt.Sprintf("-- Aegis Auth\n-- Version: %s\n-- Description: %s\n\n%s",
			migration.Version,
			migration.Description,
			migration.Up,
		)
		if err := os.WriteFile(filename, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to write auth migration: %w", err)
		}
		migrationCounter++
	}

	if e.authOnly {
		return nil
	}

	for _, plugin := range e.plugins {
		pluginMigrations := e.getPluginMigrations(plugin)
		for _, migration := range pluginMigrations {
			filename := filepath.Join(e.outputDir, fmt.Sprintf("%03d_aegis_%s_%s.sql", migrationCounter, plugin.Name(), migration.Version))
			content := fmt.Sprintf("-- Aegis Plugin: %s\n-- Version: %s\n-- Description: %s\n\n%s",
				plugin.Name(),
				migration.Version,
				migration.Description,
				migration.Up,
			)
			if err := os.WriteFile(filename, []byte(content), 0600); err != nil {
				return fmt.Errorf("failed to write %s migration: %w", plugin.Name(), err)
			}
			migrationCounter++
		}
	}

	return nil
}

// exportGoose exports migrations in Goose format
func (e *Exporter) exportGoose() error {
	authMigrations, err := e.getAuthMigrations()
	if err != nil {
		return err
	}

	migrationCounter := 1
	for _, migration := range authMigrations {
		filename := filepath.Join(e.outputDir, fmt.Sprintf("%05d_aegis_auth_%s.sql", migrationCounter, migration.Version))
		content := fmt.Sprintf("-- +goose Up\n-- +goose StatementBegin\n%s\n-- +goose StatementEnd\n\n-- +goose Down\n-- +goose StatementBegin\n%s\n-- +goose StatementEnd\n",
			migration.Up,
			migration.Down,
		)
		if err := os.WriteFile(filename, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to write auth migration: %w", err)
		}
		migrationCounter++
	}

	if e.authOnly {
		return nil
	}

	for _, plugin := range e.plugins {
		pluginMigrations := e.getPluginMigrations(plugin)
		for _, migration := range pluginMigrations {
			filename := filepath.Join(e.outputDir, fmt.Sprintf("%05d_aegis_%s_%s.sql", migrationCounter, plugin.Name(), migration.Version))
			content := fmt.Sprintf("-- +goose Up\n-- +goose StatementBegin\n%s\n-- +goose StatementEnd\n\n-- +goose Down\n-- +goose StatementBegin\n%s\n-- +goose StatementEnd\n",
				migration.Up,
				migration.Down,
			)
			if err := os.WriteFile(filename, []byte(content), 0600); err != nil {
				return fmt.Errorf("failed to write %s migration: %w", plugin.Name(), err)
			}
			migrationCounter++
		}
	}

	return nil
}

// exportGolangMigrate exports migrations in golang-migrate format
func (e *Exporter) exportGolangMigrate() error {
	authMigrations, err := e.getAuthMigrations()
	if err != nil {
		return err
	}

	migrationCounter := 1
	for _, migration := range authMigrations {
		baseName := fmt.Sprintf("%06d_aegis_auth_%s", migrationCounter, migration.Version)

		// UP migration
		upFile := filepath.Join(e.outputDir, baseName+".up.sql")
		if err := os.WriteFile(upFile, []byte(migration.Up), 0600); err != nil {
			return fmt.Errorf("failed to write auth up migration: %w", err)
		}

		// DOWN migration
		downFile := filepath.Join(e.outputDir, baseName+".down.sql")
		if err := os.WriteFile(downFile, []byte(migration.Down), 0600); err != nil {
			return fmt.Errorf("failed to write auth down migration: %w", err)
		}

		migrationCounter++
	}

	if e.authOnly {
		return nil
	}

	for _, plugin := range e.plugins {
		pluginMigrations := e.getPluginMigrations(plugin)
		for _, migration := range pluginMigrations {
			baseName := fmt.Sprintf("%06d_aegis_%s_%s", migrationCounter, plugin.Name(), migration.Version)

			// UP migration
			upFile := filepath.Join(e.outputDir, baseName+".up.sql")
			if err := os.WriteFile(upFile, []byte(migration.Up), 0600); err != nil {
				return fmt.Errorf("failed to write %s up migration: %w", plugin.Name(), err)
			}

			// DOWN migration
			downFile := filepath.Join(e.outputDir, baseName+".down.sql")
			if err := os.WriteFile(downFile, []byte(migration.Down), 0600); err != nil {
				return fmt.Errorf("failed to write %s down migration: %w", plugin.Name(), err)
			}

			migrationCounter++
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

## Running Migrations

### Using Goose

`+"```bash"+`
goose -dir %s %s up
`+"```"+`

### Using golang-migrate

`+"```bash"+`
migrate -path %s -database "%s://localhost/mydb" up
`+"```"+`

### Using Atlas

`+"```bash"+`
atlas migrate apply --dir "file://%s" --url "%s://localhost/mydb"
`+"```"+`

## Files

`, e.dialect, e.format, e.authOnly, strings.Join(pluginNames, ", "),
		e.outputDir, e.dialect,
		e.outputDir, e.dialect,
		e.outputDir, e.dialect)

	files, err := os.ReadDir(e.outputDir)
	_ = err
	sqlFiles := make([]string, 0, len(files))
	for _, file := range files {
		name := file.Name()
		if filepath.Ext(name) != ".sql" {
			continue
		}
		isUpDown := strings.HasSuffix(name, ".up.sql") || strings.HasSuffix(name, ".down.sql")
		if e.format == FormatGolangMigrate && !isUpDown {
			continue
		}
		if e.format != FormatGolangMigrate && isUpDown {
			continue
		}
		sqlFiles = append(sqlFiles, name)
	}
	sort.Strings(sqlFiles)

	for _, file := range sqlFiles {
		content += fmt.Sprintf("- `%s`\n", file)
	}

	content += `
## Custom Modifications

You can freely modify these migrations to fit your needs:

1. Add custom columns to tables
2. Add your own tables
3. Modify indexes
4. Add constraints

Just maintain the core structure needed by Aegis.

## Plugin Information

`
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

	return e.writeFile("README.md", content)
}

func (e *Exporter) writeFile(filename, content string) error {
	path := filepath.Join(e.outputDir, filename)
	return os.WriteFile(path, []byte(content), 0600)
}

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

// knownPluginOffsets maps normalized plugin names (hyphens → underscores) to
// their reserved migration base offset. These are stable and will never change.
//
// Block layout (each plugin owns up to 99 migrations in its block):
//
//	auth:          1   – 99   (offset 0, always written as-is)
//	admin:         101 – 199  (offset 100)
//	email_otp:     201 – 299  (offset 200)
//	jwt:           301 – 399  (offset 300)
//	oauth:         401 – 499  (offset 400)
//	organizations: 501 – 599  (offset 500)
//	sms:           601 – 699  (offset 600)
//
// External plugins must implement plugins.MigrationOffsetProvider and choose an
// offset starting at 1000 or higher that does not overlap with this table.
var knownPluginOffsets = map[string]int{
	"admin":         100,
	"email_otp":     200,
	"jwt":           300,
	"oauth":         400,
	"organizations": 500,
	"sms":           600,
}

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

// resolveOffset returns the stable base offset for a known aegis plugin.
// Returns an error for any plugin not registered in knownPluginOffsets.
func resolveOffset(plugin plugins.Plugin) (int, error) {
	name := strings.ReplaceAll(plugin.Name(), "-", "_")
	if offset, ok := knownPluginOffsets[name]; ok {
		return offset, nil
	}
	return 0, fmt.Errorf("plugin %q is not a known aegis plugin and cannot be exported via CLI", plugin.Name())
}

// validateOffsets checks that no two plugin sources will produce overlapping
// file numbers. It resolves every plugin's offset, computes each migration's
// actual file number (offset + version), and errors before any file is written
// if a collision is found.
//
// Auth migrations always occupy numbers 1–99 and are checked against plugins too.
func (e *Exporter) validateOffsets() error {
	// file number → plugin name that claims it
	claimed := make(map[int]string)

	// Reserve auth numbers
	authMigrations, err := e.getAuthMigrations()
	if err != nil {
		return err
	}
	for _, m := range authMigrations {
		claimed[m.Number] = "auth"
	}

	if e.authOnly {
		return nil
	}

	for _, plugin := range e.plugins {
		offset, err := resolveOffset(plugin)
		if err != nil {
			return err
		}
		for _, m := range e.getPluginMigrations(plugin) {
			num := offset + m.Number
			if owner, collision := claimed[num]; collision {
				return fmt.Errorf(
					"migration number conflict: plugin %q and %q both produce file number %d "+
						"(offset %d + version %d) — change one plugin's MigrationBaseOffset",
					plugin.Name(), owner, num, offset, m.Number,
				)
			}
			claimed[num] = plugin.Name()
		}
	}

	return nil
}

// Export writes migrations to disk in the specified format.
//
// All files are placed directly in OutputDir (flat layout) using a stable
// numeric prefix: base_offset + local_migration_version. This ensures that
// adding new migrations to auth or any plugin never renumbers another plugin's
// files, and that external plugins can coexist without central registration.
//
// Export validates for offset collisions before writing any files.
func (e *Exporter) Export() error {
	if err := e.validateOffsets(); err != nil {
		return fmt.Errorf("offset validation: %w", err)
	}

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

	return exportErr
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
	authMigrations, err := e.getAuthMigrations()
	if err != nil {
		return err
	}

	for _, m := range authMigrations {
		name := fmt.Sprintf("%05d_aegis_auth_%s.sql", m.Number, m.Version)
		content := fmt.Sprintf("-- Aegis Auth\n-- Version: %s\n-- Description: %s\n\n%s",
			m.Version, m.Description, m.Up)
		if err := os.WriteFile(filepath.Join(e.outputDir, name), []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to write auth migration: %w", err)
		}
	}

	if e.authOnly {
		return nil
	}

	for _, plugin := range e.plugins {
		offset, err := resolveOffset(plugin)
		if err != nil {
			return err
		}
		pluginName := strings.ReplaceAll(plugin.Name(), "-", "_")
		for _, m := range e.getPluginMigrations(plugin) {
			num := offset + m.Number
			name := fmt.Sprintf("%05d_aegis_%s_%s.sql", num, pluginName, m.Version)
			content := fmt.Sprintf("-- Aegis Plugin: %s\n-- Version: %s\n-- Description: %s\n\n%s",
				plugin.Name(), m.Version, m.Description, m.Up)
			if err := os.WriteFile(filepath.Join(e.outputDir, name), []byte(content), 0600); err != nil {
				return fmt.Errorf("failed to write %s migration: %w", plugin.Name(), err)
			}
		}
	}

	return nil
}

// exportGoose exports migrations in Goose format into a flat directory.
// Goose uses 5-digit numeric prefixes.
func (e *Exporter) exportGoose() error {
	authMigrations, err := e.getAuthMigrations()
	if err != nil {
		return err
	}

	for _, m := range authMigrations {
		name := fmt.Sprintf("%05d_aegis_auth_%s.sql", m.Number, m.Version)
		content := fmt.Sprintf(
			"-- +goose Up\n-- +goose StatementBegin\n%s\n-- +goose StatementEnd\n\n-- +goose Down\n-- +goose StatementBegin\n%s\n-- +goose StatementEnd\n",
			m.Up, m.Down)
		if err := os.WriteFile(filepath.Join(e.outputDir, name), []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to write auth migration: %w", err)
		}
	}

	if e.authOnly {
		return nil
	}

	for _, plugin := range e.plugins {
		offset, err := resolveOffset(plugin)
		if err != nil {
			return err
		}
		pluginName := strings.ReplaceAll(plugin.Name(), "-", "_")
		for _, m := range e.getPluginMigrations(plugin) {
			num := offset + m.Number
			name := fmt.Sprintf("%05d_aegis_%s_%s.sql", num, pluginName, m.Version)
			content := fmt.Sprintf(
				"-- +goose Up\n-- +goose StatementBegin\n%s\n-- +goose StatementEnd\n\n-- +goose Down\n-- +goose StatementBegin\n%s\n-- +goose StatementEnd\n",
				m.Up, m.Down)
			if err := os.WriteFile(filepath.Join(e.outputDir, name), []byte(content), 0600); err != nil {
				return fmt.Errorf("failed to write %s migration: %w", plugin.Name(), err)
			}
		}
	}

	return nil
}

// exportGolangMigrate exports migrations in golang-migrate format into a flat
// directory. golang-migrate uses 6-digit numeric prefixes and paired
// .up.sql / .down.sql files.
func (e *Exporter) exportGolangMigrate() error {
	authMigrations, err := e.getAuthMigrations()
	if err != nil {
		return err
	}

	for _, m := range authMigrations {
		base := filepath.Join(e.outputDir, fmt.Sprintf("%06d_aegis_auth_%s", m.Number, m.Version))
		if err := os.WriteFile(base+".up.sql", []byte(m.Up), 0600); err != nil {
			return fmt.Errorf("failed to write auth up migration: %w", err)
		}
		if err := os.WriteFile(base+".down.sql", []byte(m.Down), 0600); err != nil {
			return fmt.Errorf("failed to write auth down migration: %w", err)
		}
	}

	if e.authOnly {
		return nil
	}

	for _, plugin := range e.plugins {
		offset, err := resolveOffset(plugin)
		if err != nil {
			return err
		}
		pluginName := strings.ReplaceAll(plugin.Name(), "-", "_")
		for _, m := range e.getPluginMigrations(plugin) {
			num := offset + m.Number
			base := filepath.Join(e.outputDir, fmt.Sprintf("%06d_aegis_%s_%s", num, pluginName, m.Version))
			if err := os.WriteFile(base+".up.sql", []byte(m.Up), 0600); err != nil {
				return fmt.Errorf("failed to write %s up migration: %w", plugin.Name(), err)
			}
			if err := os.WriteFile(base+".down.sql", []byte(m.Down), 0600); err != nil {
				return fmt.Errorf("failed to write %s down migration: %w", plugin.Name(), err)
			}
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

All files live in a single flat directory. Each source occupies a stable numeric
block so that adding new migrations to one source never renumbers another:

| Source        | Block       |
|---------------|-------------|
| auth          | 00001–00099 |
| admin         | 00101–00199 |
| email_otp     | 00201–00299 |
| jwt           | 00301–00399 |
| oauth         | 00401–00499 |
| organizations | 00501–00599 |
| sms           | 00601–00699 |
| external      | 01000+      |

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

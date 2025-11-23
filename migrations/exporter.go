package migrations

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/theinventorylib/aegis/plugins"
)

// ExportFormat represents the target migration tool format
type ExportFormat string

const (
	FormatSQL           ExportFormat = "sql"
	FormatGoose         ExportFormat = "goose"
	FormatGolangMigrate ExportFormat = "golang-migrate"
)

// Exporter handles exporting migrations to various formats
type Exporter struct {
	format    ExportFormat
	outputDir string
	coreOnly  bool
	plugins   []plugins.Plugin
}

// ExporterConfig configures the migration exporter
type ExporterConfig struct {
	Format    ExportFormat
	OutputDir string
	CoreOnly  bool
	Plugins   []plugins.Plugin
}

// NewExporter creates a new migration exporter
func NewExporter(config ExporterConfig) *Exporter {
	return &Exporter{
		format:    config.Format,
		outputDir: config.OutputDir,
		coreOnly:  config.CoreOnly,
		plugins:   config.Plugins,
	}
}

// Export writes migrations to disk in the specified format
func (e *Exporter) Export() error {
	// Create output directory
	if err := os.MkdirAll(e.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	switch e.format {
	case FormatSQL:
		return e.exportSQL()
	case FormatGoose:
		return e.exportGoose()
	case FormatGolangMigrate:
		return e.exportGolangMigrate()
	default:
		return fmt.Errorf("unsupported format: %s", e.format)
	}
}

// exportSQL exports migrations as plain SQL files
func (e *Exporter) exportSQL() error {
	// Export core migration
	filename := filepath.Join(e.outputDir, "001_aegis_core.sql")
	if err := os.WriteFile(filename, []byte(CoreMigration.Up), 0644); err != nil {
		return fmt.Errorf("failed to write core migration: %w", err)
	}

	if e.coreOnly {
		return nil
	}

	// Export plugin migrations
	for i, plugin := range e.plugins {
		migrations := plugin.GetMigrations()
		for j, migration := range migrations {
			// Calculate migration number (starting after core)
			migrationNum := (i * 100) + j + 2
			filename := filepath.Join(e.outputDir, fmt.Sprintf("%03d_aegis_%s_%s.sql", migrationNum, plugin.Name(), migration.Version))

			content := fmt.Sprintf("-- Aegis Plugin: %s\n-- Version: %s\n-- Description: %s\n\n%s",
				plugin.Name(),
				migration.Version,
				migration.Description,
				migration.Up,
			)

			if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write %s migration: %w", plugin.Name(), err)
			}
		}
	}

	return nil
}

// exportGoose exports migrations in Goose format
func (e *Exporter) exportGoose() error {
	// Export core migration
	filename := filepath.Join(e.outputDir, "00001_aegis_core.sql")
	content := fmt.Sprintf("-- +goose Up\n-- +goose StatementBegin\n%s\n-- +goose StatementEnd\n\n-- +goose Down\n-- +goose StatementBegin\n%s\n-- +goose StatementEnd\n",
		CoreMigration.Up,
		CoreMigration.Down,
	)
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write core migration: %w", err)
	}

	if e.coreOnly {
		return nil
	}

	// Export plugin migrations
	migrationCounter := 2
	for _, plugin := range e.plugins {
		migrations := plugin.GetMigrations()
		for _, migration := range migrations {
			filename := filepath.Join(e.outputDir, fmt.Sprintf("%05d_aegis_%s_%s.sql",
				migrationCounter,
				plugin.Name(),
				sanitizeFilename(migration.Description),
			))

			content := fmt.Sprintf("-- +goose Up\n-- +goose StatementBegin\n%s\n-- +goose StatementEnd\n\n-- +goose Down\n-- +goose StatementBegin\n%s\n-- +goose StatementEnd\n",
				migration.Up,
				migration.Down,
			)

			if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write %s migration: %w", plugin.Name(), err)
			}

			migrationCounter++
		}
	}

	return nil
}

// exportGolangMigrate exports migrations in golang-migrate format
func (e *Exporter) exportGolangMigrate() error {
	// Export core migration UP
	upFile := filepath.Join(e.outputDir, "000001_aegis_core.up.sql")
	if err := os.WriteFile(upFile, []byte(CoreMigration.Up), 0644); err != nil {
		return fmt.Errorf("failed to write core up migration: %w", err)
	}

	// Export core migration DOWN
	downFile := filepath.Join(e.outputDir, "000001_aegis_core.down.sql")
	if err := os.WriteFile(downFile, []byte(CoreMigration.Down), 0644); err != nil {
		return fmt.Errorf("failed to write core down migration: %w", err)
	}

	if e.coreOnly {
		return nil
	}

	// Export plugin migrations
	migrationCounter := 2
	for _, plugin := range e.plugins {
		migrations := plugin.GetMigrations()
		for _, migration := range migrations {
			baseName := fmt.Sprintf("%06d_aegis_%s_%s",
				migrationCounter,
				plugin.Name(),
				sanitizeFilename(migration.Description),
			)

			// UP migration
			upFile := filepath.Join(e.outputDir, baseName+".up.sql")
			if err := os.WriteFile(upFile, []byte(migration.Up), 0644); err != nil {
				return fmt.Errorf("failed to write %s up migration: %w", plugin.Name(), err)
			}

			// DOWN migration
			downFile := filepath.Join(e.outputDir, baseName+".down.sql")
			if err := os.WriteFile(downFile, []byte(migration.Down), 0644); err != nil {
				return fmt.Errorf("failed to write %s down migration: %w", plugin.Name(), err)
			}

			migrationCounter++
		}
	}

	return nil
}

// sanitizeFilename converts a description to a safe filename
func sanitizeFilename(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	// Remove any characters that aren't alphanumeric or underscore
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

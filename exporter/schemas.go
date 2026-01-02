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

// SchemaExporterConfig configures the schema exporter
type SchemaExporterConfig struct {
	Dialect   plugins.Dialect
	OutputDir string
	AuthOnly  bool
	Plugins   []plugins.Plugin // Plugin instances to export
}

// SchemaExporter handles exporting schemas
type SchemaExporter struct {
	dialect   plugins.Dialect
	outputDir string
	authOnly  bool
	plugins   []plugins.Plugin
}

// NewSchemaExporter creates a new schema exporter
func NewSchemaExporter(config SchemaExporterConfig) *SchemaExporter {
	return &SchemaExporter{
		dialect:   config.Dialect,
		outputDir: config.OutputDir,
		authOnly:  config.AuthOnly,
		plugins:   config.Plugins,
	}
}

// Export writes schemas to disk
func (e *SchemaExporter) Export() error {
	if err := os.MkdirAll(e.outputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Export auth schema (read from embedded files)
	if err := e.exportAuthSchema(); err != nil {
		return fmt.Errorf("export auth schema: %w", err)
	}

	// Export plugin schemas if not auth-only
	if !e.authOnly {
		for _, plugin := range e.plugins {
			if err := e.exportPluginSchema(plugin); err != nil {
				return fmt.Errorf("export plugin %s schema: %w", plugin.Name(), err)
			}
		}
	}

	// Generate README
	if err := e.generateReadme(); err != nil {
		return fmt.Errorf("generate readme: %w", err)
	}

	return nil
}

// exportAuthSchema exports the auth schema from embedded files
func (e *SchemaExporter) exportAuthSchema() error {
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
		return fmt.Errorf("unsupported dialect: %s", e.dialect)
	}

	// Get schema from auth package
	authSchema, err := auth.GetSchema(authDialect)
	if err != nil {
		return fmt.Errorf("get auth schema: %w", err)
	}

	filename := filepath.Join(e.outputDir, fmt.Sprintf("001_aegis_auth_%s.sql", e.dialect))
	header := fmt.Sprintf(`-- Aegis Auth Schema
-- Dialect: %s
-- Description: Core authentication tables for Aegis

`, e.dialect)

	if err := os.WriteFile(filename, []byte(header+authSchema.SQL), 0600); err != nil {
		return fmt.Errorf("write auth schema: %w", err)
	}

	return nil
}

// exportPluginSchema exports a plugin's schema
func (e *SchemaExporter) exportPluginSchema(plugin plugins.Plugin) error {
	schemas := plugin.GetSchemas()

	// Find schema for our dialect
	var targetSchema *plugins.Schema
	for i := range schemas {
		if schemas[i].Dialect == e.dialect {
			targetSchema = &schemas[i]
			break
		}
	}

	if targetSchema == nil {
		// Plugin doesn't support this dialect - skip
		return nil
	}

	// Generate filename based on plugin and version
	version := targetSchema.Info.Version
	if version == 0 {
		version = 1
	}

	filename := filepath.Join(e.outputDir,
		fmt.Sprintf("%03d_aegis_%s_%s.sql", version+1, plugin.Name(), e.dialect))

	header := fmt.Sprintf(`-- Aegis Plugin: %s
-- Dialect: %s
-- Version: %s
-- Description: %s

`, plugin.Name(), e.dialect, plugin.Version(), targetSchema.Info.Description)

	content := header + targetSchema.SQL

	if err := os.WriteFile(filename, []byte(content), 0600); err != nil {
		return fmt.Errorf("write plugin schema: %w", err)
	}

	return nil
}

// generateReadme creates a README for the exported schemas
func (e *SchemaExporter) generateReadme() error {
	pluginNames := make([]string, 0, len(e.plugins))
	for _, p := range e.plugins {
		pluginNames = append(pluginNames, p.Name())
	}
	sort.Strings(pluginNames)

	content := fmt.Sprintf(`# Aegis Database Schema

Complete database schema exported from Aegis authentication library.

## Configuration

- **Dialect:** %s
- **Auth Only:** %v
- **Included Plugins:** %s

## Files

`, e.dialect, e.authOnly, strings.Join(pluginNames, ", "))

	files, _ := os.ReadDir(e.outputDir)
	var sqlFiles []string
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".sql" {
			sqlFiles = append(sqlFiles, file.Name())
		}
	}
	sort.Strings(sqlFiles)

	for _, file := range sqlFiles {
		content += fmt.Sprintf("- `%s`\n", file)
	}

	content += `
## Usage

### Direct Import

You can import these schemas directly into your database:

**PostgreSQL:**
` + "```bash" + `
psql -U username -d database -f schema_file.sql
` + "```" + `

**MySQL:**
` + "```bash" + `
mysql -u username -p database < schema_file.sql
` + "```" + `

### Schema Migration Tools

These schemas can be used as a starting point for your migration tool:

- **Goose:** Rename files to match Goose format
- **golang-migrate:** Use as initial migration files
- **Atlas:** Import as baseline schema

## Customization

Feel free to modify these schemas:

1. Add custom columns to existing tables
2. Add your own tables
3. Modify indexes and constraints
4. Add database-specific optimizations

**Important:** Maintain the core structure that Aegis depends on:
- Table names and primary keys
- Foreign key relationships
- Required columns

## Schema Sources

Original schemas are available in the Aegis repository:
- Auth: ` + "`github.com/theinventorylib/aegis/auth/internal/sql/[dialect]/schema.sql`" + `
- Plugins: ` + "`github.com/theinventorylib/aegis/plugins/[name]/internal/sql/[dialect]/schema.sql`" + `

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

	return e.writeFile("README.md", content)
}

func (e *SchemaExporter) writeFile(filename, content string) error {
	path := filepath.Join(e.outputDir, filename)
	return os.WriteFile(path, []byte(content), 0600)
}

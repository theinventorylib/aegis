// Package exporter provides functionality for exporting database migrations and schemas.
package exporter

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/theinventorylib/aegis/v2/auth"
	"github.com/theinventorylib/aegis/v2/plugins"
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

// sourceMigration is one migration as it exists in the Aegis source tree,
// before a sequence number is assigned for a consumer's migrations folder.
type sourceMigration struct {
	Source      string
	Version     int
	Description string
	Up          string
	Down        string
}

// migrationPlan is one migration with the sequence number and filename base it
// will use in the output directory.
type migrationPlan struct {
	Source      string
	Version     int
	Description string
	Up          string
	Down        string
	Number      int
	Base        string
	IsNew       bool
}

// existingFile describes a migration-like file already present on disk.
type existingFile struct {
	Base        string
	Number      int
	IsAegis     bool
	Source      string
	Version     int
	Description string
}

var (
	numericPrefix       = regexp.MustCompile(`^(\d+)_`)
	sourceVersionPrefix = regexp.MustCompile(`^(\d{3})_(.+)$`)
)

// Export writes migrations to disk in the specified format.
//
// The output directory is scanned for the highest numeric prefix. Aegis
// migrations are appended after it; files that already exist are matched by
// source and source version (or by description for files written by older
// exporters) and are never renumbered, rewritten, or deleted. Goose,
// golang-migrate, and plain SQL runners decide which files to apply.
func (e *Exporter) Export() error {
	if err := os.MkdirAll(e.outputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	migrations, err := e.collectAllMigrations()
	if err != nil {
		return err
	}

	plan, err := e.planMigrations(migrations)
	if err != nil {
		return err
	}

	switch e.format {
	case FormatSQL:
		err = e.writeSQL(plan)
	case FormatGoose:
		err = e.writeGoose(plan)
	case FormatGolangMigrate:
		err = e.writeGolangMigrate(plan)
	default:
		return fmt.Errorf("unsupported format: %s", e.format)
	}
	if err != nil {
		return err
	}

	return e.generateReadme()
}

// collectAllMigrations assembles every migration that should be written.
//
// Ordering:
//  1. Auth migrations, sorted by their internal version number.
//  2. Plugin migrations, plugins sorted alphabetically by name, each plugin's
//     migrations sorted by their internal version number.
func (e *Exporter) collectAllMigrations() ([]sourceMigration, error) {
	var result []sourceMigration

	authMigrations, err := e.getAuthMigrations()
	if err != nil {
		return nil, err
	}
	result = append(result, authMigrations...)

	if !e.authOnly {
		pluginList := make([]plugins.Plugin, len(e.plugins))
		copy(pluginList, e.plugins)
		sort.Slice(pluginList, func(i, j int) bool {
			return pluginList[i].Name() < pluginList[j].Name()
		})
		for _, p := range pluginList {
			result = append(result, e.getPluginMigrations(p)...)
		}
	}

	return result, nil
}

// planMigrations assigns a sequence number to every source migration. Existing
// files are reused; only migrations that are not already present are appended.
func (e *Exporter) planMigrations(migrations []sourceMigration) ([]migrationPlan, error) {
	existing, err := e.scanExisting()
	if err != nil {
		return nil, err
	}

	maxExisting := 0
	used := make(map[int]string)
	byVersion := make(map[string]existingFile)
	byDescription := make(map[string]existingFile)
	for _, f := range existing {
		if f.Number > maxExisting {
			maxExisting = f.Number
		}
		if prior, ok := used[f.Number]; ok && prior != f.Base {
			return nil, fmt.Errorf("migration number %d is used by both %s and %s", f.Number, prior, f.Base)
		}
		used[f.Number] = f.Base
		if !f.IsAegis || f.Source == "" {
			continue
		}
		if f.Version > 0 {
			byVersion[sourceVersionKey(f.Source, f.Version)] = f
		}
		if f.Description != "" {
			byDescription[sourceDescriptionKey(f.Source, f.Description)] = f
		}
	}

	next := maxExisting + 1
	if next < 1 {
		next = 1
	}

	plan := make([]migrationPlan, 0, len(migrations))
	seen := make(map[string]bool, len(migrations))
	for _, m := range migrations {
		key := sourceVersionKey(m.Source, m.Version)
		if seen[key] {
			return nil, fmt.Errorf("duplicate migration %s", key)
		}
		seen[key] = true

		if f, ok := byVersion[key]; ok {
			plan = append(plan, planFromExisting(m, f))
			continue
		}
		// Files from exports before the source version was part of the
		// filename are matched by source and description.
		if f, ok := byDescription[sourceDescriptionKey(m.Source, m.Description)]; ok {
			plan = append(plan, planFromExisting(m, f))
			continue
		}

		plan = append(plan, migrationPlan{
			Source:      m.Source,
			Version:     m.Version,
			Description: m.Description,
			Up:          m.Up,
			Down:        m.Down,
			Number:      next,
			Base:        e.migrationBase(next, m),
			IsNew:       true,
		})
		next++
	}

	return plan, nil
}

func planFromExisting(m sourceMigration, f existingFile) migrationPlan {
	return migrationPlan{
		Source:      m.Source,
		Version:     m.Version,
		Description: m.Description,
		Up:          m.Up,
		Down:        m.Down,
		Number:      f.Number,
		Base:        f.Base,
	}
}

// scanExisting reads the output directory and classifies migration files.
// Up/down pairs share one base and are reported once.
func (e *Exporter) scanExisting() ([]existingFile, error) {
	entries, err := os.ReadDir(e.outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading output directory: %w", err)
	}
	sources := e.knownSources()
	seen := make(map[string]bool)
	var out []existingFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		base, ok := stripMigrationExtension(entry.Name())
		if !ok || seen[base] {
			continue
		}
		match := numericPrefix.FindStringSubmatch(base)
		if match == nil {
			continue
		}
		number, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		seen[base] = true

		f := existingFile{Base: base, Number: number}
		rest := strings.TrimPrefix(base, match[0])
		if strings.HasPrefix(rest, "aegis_") {
			f.IsAegis = true
			body := strings.TrimPrefix(rest, "aegis_")
			f.Source, body = matchSource(body, sources)
			if f.Source != "" {
				if sm := sourceVersionPrefix.FindStringSubmatch(body); sm != nil {
					if version, convErr := strconv.Atoi(sm[1]); convErr == nil {
						f.Version = version
						f.Description = sm[2]
					}
				}
				if f.Description == "" {
					f.Description = body
				}
			} else {
				f.Description = body
			}
		}
		out = append(out, f)
	}
	return out, nil
}

func (e *Exporter) knownSources() []string {
	sources := []string{"auth"}
	if e.authOnly {
		return sources
	}
	for _, p := range e.plugins {
		sources = append(sources, p.Name())
	}
	return sources
}

// matchSource splits a filename body into the plugin/auth source and the rest.
// Plugin names may contain hyphens while filenames use underscores.
func matchSource(rest string, sources []string) (source, remainder string) {
	sort.Slice(sources, func(i, j int) bool {
		si := strings.ReplaceAll(sources[i], "-", "_")
		sj := strings.ReplaceAll(sources[j], "-", "_")
		return len(si) > len(sj)
	})
	for _, s := range sources {
		slug := strings.ReplaceAll(s, "-", "_")
		if strings.HasPrefix(rest, slug+"_") {
			return s, strings.TrimPrefix(rest, slug+"_")
		}
	}
	return "", rest
}

func (e *Exporter) migrationBase(number int, m sourceMigration) string {
	source := strings.ReplaceAll(m.Source, "-", "_")
	if e.format == FormatGolangMigrate {
		return fmt.Sprintf("%06d_aegis_%s_%03d_%s", number, source, m.Version, m.Description)
	}
	return fmt.Sprintf("%05d_aegis_%s_%03d_%s", number, source, m.Version, m.Description)
}

func sourceVersionKey(source string, version int) string {
	return source + "/" + strconv.Itoa(version)
}

func sourceDescriptionKey(source, description string) string {
	return source + "\x00" + description
}

// stripMigrationExtension removes the suffix a migration tool appends and
// returns the stable base name. Aegis uses the same base for SQL, Goose and
// golang-migrate files.
func stripMigrationExtension(name string) (string, bool) {
	switch {
	case strings.HasSuffix(name, ".up.sql"):
		return strings.TrimSuffix(name, ".up.sql"), true
	case strings.HasSuffix(name, ".down.sql"):
		return strings.TrimSuffix(name, ".down.sql"), true
	case strings.HasSuffix(name, ".sql"):
		return strings.TrimSuffix(name, ".sql"), true
	default:
		return "", false
	}
}

// writeSQL exports migrations as plain SQL files into a flat directory.
func (e *Exporter) writeSQL(plan []migrationPlan) error {
	for _, p := range plan {
		if !p.IsNew || e.fileExists(p.Base+".sql") {
			continue
		}
		content := fmt.Sprintf("-- Aegis: %s\n-- Version: %d\n-- Description: %s\n\n%s",
			p.Source, p.Version, p.Description, p.Up)
		if err := os.WriteFile(filepath.Join(e.outputDir, p.Base+".sql"), []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to write %s migration: %w", p.Source, err)
		}
	}
	return nil
}

// writeGoose exports migrations in Goose format into a flat directory.
func (e *Exporter) writeGoose(plan []migrationPlan) error {
	for _, p := range plan {
		if !p.IsNew || e.fileExists(p.Base+".sql") {
			continue
		}
		content := fmt.Sprintf(
			"-- +goose Up\n-- +goose StatementBegin\n%s\n-- +goose StatementEnd\n\n-- +goose Down\n-- +goose StatementBegin\n%s\n-- +goose StatementEnd\n",
			p.Up, p.Down)
		if err := os.WriteFile(filepath.Join(e.outputDir, p.Base+".sql"), []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to write %s migration: %w", p.Source, err)
		}
	}
	return nil
}

// writeGolangMigrate exports migrations in golang-migrate format into a flat
// directory. golang-migrate uses numeric prefixes and paired .up.sql /
// .down.sql files.
func (e *Exporter) writeGolangMigrate(plan []migrationPlan) error {
	for _, p := range plan {
		if !p.IsNew {
			continue
		}
		if !e.fileExists(p.Base + ".up.sql") {
			if err := os.WriteFile(filepath.Join(e.outputDir, p.Base+".up.sql"), []byte(p.Up), 0600); err != nil {
				return fmt.Errorf("failed to write %s up migration: %w", p.Source, err)
			}
		}
		if !e.fileExists(p.Base + ".down.sql") {
			if err := os.WriteFile(filepath.Join(e.outputDir, p.Base+".down.sql"), []byte(p.Down), 0600); err != nil {
				return fmt.Errorf("failed to write %s down migration: %w", p.Source, err)
			}
		}
	}
	return nil
}

func (e *Exporter) fileExists(name string) bool {
	_, err := os.Stat(filepath.Join(e.outputDir, name))
	return err == nil
}

func (e *Exporter) getAuthMigrations() ([]sourceMigration, error) {
	allMigrations, err := auth.GetMigrations(e.dialect)
	if err != nil {
		return nil, fmt.Errorf("get auth migrations: %w", err)
	}

	migrations := make([]sourceMigration, 0, len(allMigrations))
	for _, am := range allMigrations {
		migrations = append(migrations, sourceMigration{
			Source:      "auth",
			Version:     am.Version,
			Description: am.Description,
			Up:          am.Up,
			Down:        am.Down,
		})
	}
	return migrations, nil
}

func (e *Exporter) getPluginMigrations(plugin plugins.Plugin) []sourceMigration {
	pluginMigrations := plugin.GetMigrations()

	migrations := make([]sourceMigration, 0, len(pluginMigrations))
	for _, pm := range pluginMigrations {
		migrations = append(migrations, sourceMigration{
			Source:      plugin.Name(),
			Version:     pm.Version,
			Description: pm.Description,
			Up:          pm.Up,
			Down:        pm.Down,
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations
}

func (e *Exporter) generateReadme() error {
	pluginNames := make([]string, 0, len(e.plugins))
	for _, p := range e.plugins {
		pluginNames = append(pluginNames, p.Name())
	}
	sort.Strings(pluginNames)

	var content strings.Builder
	fmt.Fprintf(&content, `# Aegis Migrations

These migrations were exported from Aegis authentication library.

## Configuration

- **Dialect:** %s
- **Format:** %s
- **Auth Only:** %v
- **Included Plugins:** %s

## Numbering Scheme

Aegis scans the highest numeric prefix in this directory and appends its
migrations after it. If the directory already contains your own migrations,
Aegis starts after the highest one. Existing Aegis files are matched by source
and source version (or by description for files written by older exporters)
and are never renumbered, rewritten, or deleted.

The migration tool you use decides what to apply. Aegis does not run
migrations.

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
		content.WriteString("| Plugin | Version | Description |\n")
		content.WriteString("|--------|---------|-------------|\n")
		for _, p := range e.plugins {
			fmt.Fprintf(&content, "| %s | %s | %s |\n", p.Name(), p.Version(), p.Description())
		}
	} else {
		content.WriteString("No plugins included in this export.\n")
	}

	content.WriteString(`
## Migration Sources

The original migrations are available at:
- Auth: github.com/theinventorylib/aegis/v2/auth/migrations/[dialect]/
- Plugins: github.com/theinventorylib/aegis/v2/plugins/[name]/migrations/[dialect]/
`)

	readmePath := filepath.Join(e.outputDir, "README.md")
	if data, err := os.ReadFile(readmePath); err == nil { // #nosec G304 -- caller-selected output directory
		// Never replace a README the user wrote. Older Aegis exports used the
		// same filename, so refresh those (and only those).
		if !strings.HasPrefix(string(data), "# Aegis Migrations") {
			return nil
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	return os.WriteFile(readmePath, []byte(content.String()), 0600)
}

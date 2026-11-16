// Package exporter provides functionality for exporting database migrations and schemas.
package exporter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

const (
	manifestName    = ".aegis-migrations.json"
	manifestVersion = 1
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

	// StartVersion is the first sequence number to assign on a first export
	// into a directory that already contains application migrations. Zero
	// means "assign automatically", which is only allowed for an empty
	// directory or one containing Aegis migrations only.
	StartVersion int
}

// Exporter handles exporting migrations to various formats.
type Exporter struct {
	format       ExportFormat
	outputDir    string
	dialect      plugins.Dialect
	authOnly     bool
	plugins      []plugins.Plugin
	startVersion int

	// Warnings collects non-fatal issues found during Export, such as stale
	// Aegis files left in the output directory.
	Warnings []string
}

// NewExporter creates a new migration exporter with the provided configuration.
func NewExporter(config Config) *Exporter {
	return &Exporter{
		format:       config.Format,
		outputDir:    config.OutputDir,
		dialect:      config.Dialect,
		authOnly:     config.AuthOnly,
		plugins:      config.Plugins,
		startVersion: config.StartVersion,
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
	ID          string
	Number      int
	Source      string
	Description string
	Up          string
	Down        string
	Base        string
	Hash        string
	IsNew       bool
}

// manifestEntry records the stable assignment for one Aegis source migration.
type manifestEntry struct {
	Number int    `json:"number"`
	Base   string `json:"base"`
	SHA256 string `json:"sha256"`
}

// manifest is the lock file that makes exports append-only. Without it a second
// export would renumber already-applied migrations, which breaks migration
// tools that track integer versions.
type manifest struct {
	Version    int                      `json:"version"`
	Migrations map[string]manifestEntry `json:"migrations"`
}

// existingFile describes a migration-like file already present on disk.
type existingFile struct {
	Base        string
	Number      int
	Source      string
	Description string
	IsAegis     bool
}

var numericPrefix = regexp.MustCompile(`^(\d+)_`)

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
	for _, m := range authMigrations {
		result = append(result, sourceMigration{
			Source:      "auth",
			Version:     m.Number,
			Description: m.Description,
			Up:          m.Up,
			Down:        m.Down,
		})
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
				result = append(result, sourceMigration{
					Source:      p.Name(),
					Version:     m.Number,
					Description: m.Description,
					Up:          m.Up,
					Down:        m.Down,
				})
			}
		}
	}

	return result, nil
}

// Export writes migrations to disk in the specified format.
//
// The output directory is append-only: once a migration has been exported it
// keeps its sequence number for every later export. New migrations are appended
// after the highest number already present (Aegis or application-owned). The
// first export into a directory that already contains application migrations
// requires Config.StartVersion so Aegis cannot silently write numbers lower
// than ones the consumer already applied.
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

	if err := e.writeManifest(plan); err != nil {
		return fmt.Errorf("write migration manifest: %w", err)
	}

	if err := e.generateReadme(); err != nil {
		return fmt.Errorf("generate readme: %w", err)
	}

	return nil
}

// planState holds the numbering context while a plan is assembled.
type planState struct {
	maxExisting int
	used        map[int]string
	aegisFiles  map[string]existingFile
	plan        []migrationPlan
	seen        map[int]string
	ids         map[string]bool
}

func newPlanState(existing []existingFile) (*planState, error) {
	st := &planState{
		used:       make(map[int]string),
		aegisFiles: make(map[string]existingFile),
		seen:       make(map[int]string),
		ids:        make(map[string]bool),
	}
	for _, f := range existing {
		if f.Number > st.maxExisting {
			st.maxExisting = f.Number
		}
		if prior, ok := st.used[f.Number]; ok && prior != f.Base {
			return nil, fmt.Errorf("migration number %d is used by both %s and %s", f.Number, prior, f.Base)
		}
		st.used[f.Number] = f.Base
		if f.IsAegis {
			st.aegisFiles[f.Source+"\x00"+f.Description] = f
		}
	}
	return st, nil
}

func (st *planState) reserveID(id string) error {
	if st.ids[id] {
		return fmt.Errorf("duplicate migration ID %q", id)
	}
	st.ids[id] = true
	return nil
}

func (st *planState) add(p migrationPlan) error {
	if prior, ok := st.seen[p.Number]; ok && prior != p.Base {
		return fmt.Errorf("migration %s reuses sequence number %d already used by %s", p.ID, p.Number, prior)
	}
	st.seen[p.Number] = p.Base
	st.plan = append(st.plan, p)
	return nil
}

// planMigrations decides the sequence number for every source migration.
func (e *Exporter) planMigrations(migrations []sourceMigration) ([]migrationPlan, error) {
	if e.startVersion < 0 {
		return nil, fmt.Errorf("start version must not be negative")
	}

	existing, err := e.scanExisting()
	if err != nil {
		return nil, err
	}
	mf, err := e.loadManifest()
	if err != nil {
		return nil, err
	}
	st, err := newPlanState(existing)
	if err != nil {
		return nil, err
	}

	next, err := e.firstNumber(st, mf)
	if err != nil {
		return nil, err
	}

	for _, m := range migrations {
		id := migrationID(m)
		if err := st.reserveID(id); err != nil {
			return nil, err
		}
		hash := migrationHash(m)

		if mf != nil {
			if entry, ok := mf.Migrations[id]; ok {
				if err := e.planManifestEntry(st, m, id, entry, hash); err != nil {
					return nil, err
				}
				continue
			}
		} else if len(st.aegisFiles) > 0 {
			if f, ok := st.aegisFiles[m.Source+"\x00"+m.Description]; ok {
				if err := st.add(migrationPlan{
					ID: id, Number: f.Number, Source: m.Source, Description: m.Description,
					Up: m.Up, Down: m.Down, Base: f.Base, Hash: hash,
				}); err != nil {
					return nil, err
				}
				continue
			}
		}

		// A first export into a folder with application migrations must not
		// write a lower number unless the caller chose the start explicitly.
		if mf == nil && len(st.aegisFiles) == 0 && e.startVersion == 0 && st.maxExisting > 0 {
			return nil, fmt.Errorf(
				"output directory already contains migrations up to %05d; re-run with --start-version %d (or export into an empty directory)",
				st.maxExisting, st.maxExisting+1)
		}
		if prior, ok := st.used[next]; ok && prior != "" {
			return nil, fmt.Errorf("sequence number %d is already used by %s", next, prior)
		}
		base := e.migrationBase(next, m.Source, m.Description)
		if err := st.add(migrationPlan{
			ID: id, Number: next, Source: m.Source, Description: m.Description,
			Up: m.Up, Down: m.Down, Base: base, Hash: hash, IsNew: true,
		}); err != nil {
			return nil, err
		}
		next++
	}

	// Report files that belong to no current migration, but never delete them:
	// an applied migration whose source disappeared may still be needed by an
	// older deployment.
	planned := make(map[string]bool, len(st.plan))
	for _, p := range st.plan {
		planned[p.Base] = true
	}
	for _, f := range existing {
		if f.IsAegis && !planned[f.Base] {
			e.Warnings = append(e.Warnings,
				fmt.Sprintf("stale Aegis migration file %q is not part of the current export; it was left in place", f.Base))
		}
	}

	return st.plan, nil
}

// firstNumber resolves the first sequence number for new migrations.
func (e *Exporter) firstNumber(st *planState, mf *manifest) (int, error) {
	if e.startVersion > 0 && e.startVersion <= st.maxExisting {
		return 0, fmt.Errorf(
			"start version %d collides with an existing migration numbered %05d; choose a value greater than %d",
			e.startVersion, st.maxExisting, st.maxExisting)
	}

	next := st.maxExisting + 1
	if next < 1 {
		next = 1
	}
	if mf != nil {
		next = maxAssigned(mf, st.maxExisting) + 1
		if e.startVersion > 0 && e.startVersion != next {
			return 0, fmt.Errorf(
				"--start-version only applies to the first export; %s already records the next append position %d",
				manifestName, next)
		}
		return next, nil
	}

	if e.startVersion == 0 {
		return next, nil
	}
	if len(st.aegisFiles) == 0 {
		return e.startVersion, nil
	}
	if e.startVersion > next {
		return e.startVersion, nil
	}
	return next, nil
}

func (e *Exporter) planManifestEntry(st *planState, m sourceMigration, id string, entry manifestEntry, hash string) error {
	if entry.SHA256 != "" && entry.SHA256 != hash {
		return fmt.Errorf(
			"migration %s changed after it was exported; released migrations must be immutable — add a new migration instead",
			id)
	}
	if prior, ok := st.used[entry.Number]; ok && prior != entry.Base {
		return fmt.Errorf("migration %s collides with existing file %s at sequence number %d", id, prior, entry.Number)
	}
	return st.add(migrationPlan{
		ID: id, Number: entry.Number, Source: m.Source, Description: m.Description,
		Up: m.Up, Down: m.Down, Base: entry.Base, Hash: hash,
	})
}

// scanExisting reads the output directory and classifies migration files. The
// directory may contain Aegis files from an older exporter (no manifest), which
// are adopted by source+description matching in planMigrations.
func (e *Exporter) scanExisting() ([]existingFile, error) {
	entries, err := os.ReadDir(e.outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading output directory: %w", err)
	}
	sources := e.knownSources()
	var out []existingFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		base, ok := stripMigrationExtension(name)
		if !ok {
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
		rest := strings.TrimPrefix(base, match[0])
		f := existingFile{Base: base, Number: number}
		if strings.HasPrefix(rest, "aegis_") {
			f.IsAegis = true
			f.Source, f.Description = parseAegisName(strings.TrimPrefix(rest, "aegis_"), sources)
		}
		out = append(out, f)
	}
	return out, nil
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

// parseAegisName splits "<source>_<description>" using the known source names.
// Plugin names may contain hyphens, while filenames use underscores, so match
// on the sanitized form and return the original name for manifest keys.
func parseAegisName(rest string, sources []string) (source, description string) {
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

func (e *Exporter) migrationBase(number int, source, description string) string {
	source = strings.ReplaceAll(source, "-", "_")
	if e.format == FormatGolangMigrate {
		return fmt.Sprintf("%06d_aegis_%s_%s", number, source, description)
	}
	return fmt.Sprintf("%05d_aegis_%s_%s", number, source, description)
}

func migrationID(m sourceMigration) string {
	return fmt.Sprintf("%s/%03d", m.Source, m.Version)
}

func migrationHash(m sourceMigration) string {
	h := sha256.New()
	_, _ = h.Write([]byte(m.Up))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(m.Down))
	return hex.EncodeToString(h.Sum(nil))
}

func maxAssigned(mf *manifest, fallback int) int {
	max := fallback
	for _, entry := range mf.Migrations {
		if entry.Number > max {
			max = entry.Number
		}
	}
	return max
}

func (e *Exporter) loadManifest() (*manifest, error) {
	path := filepath.Join(e.outputDir, manifestName)
	data, err := os.ReadFile(path) // #nosec G304 -- path is the caller-selected output directory
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read migration manifest: %w", err)
	}
	var mf manifest
	if err := json.Unmarshal(data, &mf); err != nil {
		return nil, fmt.Errorf("parse %s: %w", manifestName, err)
	}
	if mf.Version > manifestVersion {
		return nil, fmt.Errorf("unsupported %s version %d (this CLI supports %d)", manifestName, mf.Version, manifestVersion)
	}
	if mf.Migrations == nil {
		mf.Migrations = make(map[string]manifestEntry)
	}
	return &mf, nil
}

func (e *Exporter) writeManifest(plan []migrationPlan) error {
	existing, err := e.loadManifest()
	if err != nil {
		return err
	}
	mf := &manifest{Version: manifestVersion, Migrations: make(map[string]manifestEntry)}
	if existing != nil {
		// Keep orphan entries: removing an assignment could let a later
		// export reuse a number that an old deployment already applied.
		for id, entry := range existing.Migrations {
			mf.Migrations[id] = entry
		}
	}
	for _, p := range plan {
		mf.Migrations[p.ID] = manifestEntry{Number: p.Number, Base: p.Base, SHA256: p.Hash}
	}
	data, err := json.MarshalIndent(mf, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(e.outputDir, manifestName), data, 0600)
}

// writeSQL exports migrations as plain SQL files into a flat directory.
func (e *Exporter) writeSQL(plan []migrationPlan) error {
	for _, p := range plan {
		if !p.IsNew && e.fileExists(p.Base+".sql") {
			continue
		}
		content := fmt.Sprintf("-- Aegis: %s\n-- Version: %s\n-- Description: %s\n\n%s",
			p.Source, p.Description, p.Description, p.Up)
		if err := os.WriteFile(filepath.Join(e.outputDir, p.Base+".sql"), []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to write %s migration: %w", p.Source, err)
		}
	}
	return nil
}

// writeGoose exports migrations in Goose format into a flat directory.
func (e *Exporter) writeGoose(plan []migrationPlan) error {
	for _, p := range plan {
		if !p.IsNew && e.fileExists(p.Base+".sql") {
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
		if p.IsNew || !e.fileExists(p.Base+".up.sql") {
			if err := os.WriteFile(filepath.Join(e.outputDir, p.Base+".up.sql"), []byte(p.Up), 0600); err != nil {
				return fmt.Errorf("failed to write %s up migration: %w", p.Source, err)
			}
		}
		if p.IsNew || !e.fileExists(p.Base+".down.sql") {
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

Migrations are numbered sequentially after the highest migration already in
this directory. The assignment is recorded in `+"`%s`"+`; keep that file in
version control. Re-running "aegis export" appends new Aegis migrations and
never renumbers or rewrites ones that were already exported.

If you exported Aegis before this manifest existed, the exporter adopts the
existing `+"`_aegis_`"+` files on the next run. If this directory already contains
your own migrations, the first export requires `+"`--start-version N`"+` with a
number greater than your latest migration.

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
		manifestName,
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

	if len(e.Warnings) > 0 {
		content.WriteString("\n## Warnings\n\n")
		for _, warning := range e.Warnings {
			fmt.Fprintf(&content, "- %s\n", warning)
		}
	}

	return os.WriteFile(filepath.Join(e.outputDir, "README.md"), []byte(content.String()), 0600)
}

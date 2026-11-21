package exporter

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theinventorylib/aegis/v2/auth"
	"github.com/theinventorylib/aegis/v2/plugins"
	"github.com/theinventorylib/aegis/v2/router"
)

type testPlugin struct {
	name       string
	migrations []plugins.Migration
}

func (p *testPlugin) Name() string        { return p.name }
func (p *testPlugin) Version() string     { return "test" }
func (p *testPlugin) Description() string { return "test plugin" }
func (p *testPlugin) Init(context.Context, plugins.Aegis) error {
	return nil
}
func (p *testPlugin) GetMigrations() []plugins.Migration { return p.migrations }
func (p *testPlugin) MountRoutes(router.Router, string)  {}
func (p *testPlugin) Dependencies() []plugins.Dependency { return nil }
func (p *testPlugin) RequiresTables() []string           { return nil }
func (p *testPlugin) ProvidesAuthMethods() []string      { return nil }

func sqlFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	var files []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	return files
}

func hasFile(files []string, name string) bool {
	for _, f := range files {
		if f == name {
			return true
		}
	}
	return false
}

func findFile(files []string, fragment string) string {
	for _, f := range files {
		if strings.Contains(f, fragment) {
			return f
		}
	}
	return ""
}

func TestExportStartsAtOneInEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	exp := NewExporter(Config{Format: FormatSQL, Dialect: plugins.DialectPostgres, OutputDir: dir, AuthOnly: true})
	if err := exp.Export(); err != nil {
		t.Fatalf("export: %v", err)
	}

	files := sqlFiles(t, dir)
	if !hasFile(files, "00001_aegis_auth_001_initial.sql") {
		t.Fatalf("first file missing; got %v", files)
	}
	if _, err := os.Stat(filepath.Join(dir, ".aegis-migrations.json")); !os.IsNotExist(err) {
		t.Fatal("the exporter must not create a manifest")
	}
}

func TestExportAutoAppendsAfterApplicationMigrations(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "00012_app.sql"), []byte("SELECT 1;"), 0600); err != nil {
		t.Fatal(err)
	}

	exp := NewExporter(Config{Format: FormatSQL, Dialect: plugins.DialectPostgres, OutputDir: dir, AuthOnly: true})
	if err := exp.Export(); err != nil {
		t.Fatalf("export: %v", err)
	}

	files := sqlFiles(t, dir)
	if !hasFile(files, "00013_aegis_auth_001_initial.sql") {
		t.Fatalf("Aegis should start at 13; got %v", files)
	}
	if findFile(files, "00001_aegis_") != "" {
		t.Fatalf("Aegis wrote a lower number: %v", files)
	}
	if !hasFile(files, "00012_app.sql") {
		t.Fatal("application migration must remain untouched")
	}
}

func TestExportIsAppendOnlyAcrossRuns(t *testing.T) {
	dir := t.TempDir()
	p := &testPlugin{
		name: "demo",
		migrations: []plugins.Migration{
			{Version: 1, Description: "first", Up: "CREATE TABLE demo (id TEXT);", Down: "DROP TABLE demo;"},
		},
	}
	cfg := Config{Format: FormatSQL, Dialect: plugins.DialectPostgres, OutputDir: dir, Plugins: []plugins.Plugin{p}}
	if err := NewExporter(cfg).Export(); err != nil {
		t.Fatalf("first export: %v", err)
	}
	first := findFile(sqlFiles(t, dir), "_aegis_demo_001_")
	if first == "" {
		t.Fatal("missing demo/001 migration")
	}

	p.migrations = append(p.migrations, plugins.Migration{
		Version: 2, Description: "second", Up: "ALTER TABLE demo ADD COLUMN name TEXT;", Down: "ALTER TABLE demo DROP COLUMN name;",
	})
	cfg.Plugins = []plugins.Plugin{p}
	if err := NewExporter(cfg).Export(); err != nil {
		t.Fatalf("second export: %v", err)
	}

	files := sqlFiles(t, dir)
	if got := findFile(files, "_aegis_demo_001_"); got != first {
		t.Fatalf("demo/001 was renumbered: %q -> %q", first, got)
	}
	if findFile(files, "_aegis_demo_002_") == "" {
		t.Fatalf("demo/002 was not appended; got %v", files)
	}
}

func TestExportAdoptsOldAegisFilesWithoutVersion(t *testing.T) {
	dir := t.TempDir()
	migrations, err := auth.GetMigrations(plugins.DialectPostgres)
	if err != nil {
		t.Fatalf("get auth migrations: %v", err)
	}
	for _, m := range migrations {
		name := filepath.Join(dir, oldAegisName(m.Version, "auth", m.Description))
		if err := os.WriteFile(name, []byte(m.Up), 0600); err != nil {
			t.Fatal(err)
		}
	}

	exp := NewExporter(Config{Format: FormatSQL, Dialect: plugins.DialectPostgres, OutputDir: dir, AuthOnly: true})
	if err := exp.Export(); err != nil {
		t.Fatalf("export: %v", err)
	}

	files := sqlFiles(t, dir)
	if len(files) != len(migrations) {
		t.Fatalf("old files should be adopted without new duplicates; got %v", files)
	}
	if findFile(files, "_aegis_auth_001_initial") != "" {
		t.Fatalf("old file was not adopted: %v", files)
	}
}

func TestExportMatchesSourceVersionWhenDescriptionChanges(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "00001_aegis_auth_001_renamed.sql"), []byte("SELECT 1;"), 0600); err != nil {
		t.Fatal(err)
	}

	exp := NewExporter(Config{Format: FormatSQL, Dialect: plugins.DialectPostgres, OutputDir: dir, AuthOnly: true})
	if err := exp.Export(); err != nil {
		t.Fatalf("export: %v", err)
	}

	files := sqlFiles(t, dir)
	if findFile(files, "_aegis_auth_001_initial") != "" {
		t.Fatalf("a description change must not create a duplicate: %v", files)
	}
	if !hasFile(files, "00001_aegis_auth_001_renamed.sql") {
		t.Fatal("existing file must remain in place")
	}
}

func TestExportAdoptsHyphenatedPluginFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "00001_aegis_email_otp_initial.sql"), []byte("SELECT 1;"), 0600); err != nil {
		t.Fatal(err)
	}
	p := &testPlugin{name: "email-otp", migrations: []plugins.Migration{
		{Version: 1, Description: "initial", Up: "SELECT 1;", Down: "SELECT 1;"},
	}}
	exp := NewExporter(Config{Format: FormatSQL, Dialect: plugins.DialectPostgres, OutputDir: dir, Plugins: []plugins.Plugin{p}})
	if err := exp.Export(); err != nil {
		t.Fatalf("export: %v", err)
	}

	files := sqlFiles(t, dir)
	if findFile(files, "_aegis_email_otp_001_initial") != "" {
		t.Fatalf("hyphenated plugin was duplicated: %v", files)
	}
	if !hasFile(files, "00001_aegis_email_otp_initial.sql") {
		t.Fatal("old hyphenated file must remain in place")
	}
}

func TestExportNeverRewritesExistingFiles(t *testing.T) {
	dir := t.TempDir()
	const original = "-- original content\n"
	if err := os.WriteFile(filepath.Join(dir, "00001_aegis_auth_001_initial.sql"), []byte(original), 0600); err != nil {
		t.Fatal(err)
	}

	exp := NewExporter(Config{Format: FormatSQL, Dialect: plugins.DialectPostgres, OutputDir: dir, AuthOnly: true})
	if err := exp.Export(); err != nil {
		t.Fatalf("export: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "00001_aegis_auth_001_initial.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Fatalf("existing file was rewritten:\n%s", data)
	}
}

func TestExportSQLFormat(t *testing.T) {
	dir := t.TempDir()
	exp := NewExporter(Config{Format: FormatSQL, Dialect: plugins.DialectPostgres, OutputDir: dir, AuthOnly: true})
	if err := exp.Export(); err != nil {
		t.Fatalf("export: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "00001_aegis_auth_001_initial.sql"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "-- Aegis: auth") || !strings.Contains(content, "-- Version: 1") {
		t.Fatalf("unexpected SQL header:\n%s", content)
	}
}

func TestExportGooseFormat(t *testing.T) {
	dir := t.TempDir()
	exp := NewExporter(Config{Format: FormatGoose, Dialect: plugins.DialectPostgres, OutputDir: dir, AuthOnly: true})
	if err := exp.Export(); err != nil {
		t.Fatalf("export: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "00001_aegis_auth_001_initial.sql"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "-- +goose Up") || !strings.Contains(content, "-- +goose Down") {
		t.Fatalf("unexpected Goose wrapper:\n%s", content)
	}
}

func TestExportGolangMigratePairs(t *testing.T) {
	dir := t.TempDir()
	exp := NewExporter(Config{
		Format: FormatGolangMigrate, Dialect: plugins.DialectSQLite, OutputDir: dir, AuthOnly: true,
	})
	if err := exp.Export(); err != nil {
		t.Fatalf("export: %v", err)
	}
	up := filepath.Join(dir, "000001_aegis_auth_001_initial.up.sql")
	down := filepath.Join(dir, "000001_aegis_auth_001_initial.down.sql")
	if _, err := os.Stat(up); err != nil {
		t.Fatalf("missing up migration: %v", err)
	}
	if _, err := os.Stat(down); err != nil {
		t.Fatalf("missing down migration: %v", err)
	}
}

func oldAegisName(version int, source, description string) string {
	return pad(version, 5) + "_aegis_" + source + "_" + description + ".sql"
}

func pad(n, width int) string {
	s := itoa(n)
	for len(s) < width {
		s = "0" + s
	}
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

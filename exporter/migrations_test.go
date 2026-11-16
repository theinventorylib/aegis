package exporter

import (
	"context"
	"encoding/json"
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

func readManifest(t *testing.T, dir string) manifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, manifestName))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var mf manifest
	if err := json.Unmarshal(data, &mf); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	return mf
}

func TestExportStartsAtOneInEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	exp := NewExporter(Config{Format: FormatSQL, Dialect: plugins.DialectPostgres, OutputDir: dir, AuthOnly: true})
	if err := exp.Export(); err != nil {
		t.Fatalf("export: %v", err)
	}

	files := sqlFiles(t, dir)
	if len(files) == 0 {
		t.Fatal("expected SQL migrations")
	}
	if files[0] != "00001_aegis_auth_initial.sql" {
		t.Fatalf("first file = %q, want 00001_aegis_auth_initial.sql", files[0])
	}
	mf := readManifest(t, dir)
	if got := mf.Migrations["auth/001"].Number; got != 1 {
		t.Fatalf("auth/001 number = %d, want 1", got)
	}
}

func TestExportRefusesLowerNumberInApplicationDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "00012_app.sql"), []byte("SELECT 1;"), 0600); err != nil {
		t.Fatal(err)
	}

	exp := NewExporter(Config{Format: FormatSQL, Dialect: plugins.DialectPostgres, OutputDir: dir, AuthOnly: true})
	err := exp.Export()
	if err == nil {
		t.Fatal("export must refuse to write lower numbers into a non-empty directory")
	}
	if !strings.Contains(err.Error(), "--start-version 13") {
		t.Fatalf("error should name the required start version, got: %v", err)
	}
}

func TestExportHonorsStartVersion(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "00012_app.sql"), []byte("SELECT 1;"), 0600); err != nil {
		t.Fatal(err)
	}

	exp := NewExporter(Config{
		Format: FormatSQL, Dialect: plugins.DialectPostgres, OutputDir: dir, AuthOnly: true, StartVersion: 13,
	})
	if err := exp.Export(); err != nil {
		t.Fatalf("export: %v", err)
	}

	files := sqlFiles(t, dir)
	found := false
	for _, f := range files {
		if strings.HasPrefix(f, "00013_aegis_auth_initial") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected Aegis migrations to start at 00013, got %v", files)
	}
	if _, err := os.Stat(filepath.Join(dir, "00001_aegis_auth_initial.sql")); !os.IsNotExist(err) {
		t.Fatal("must not write a lower-numbered Aegis migration alongside app migrations")
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
	first := readManifest(t, dir)
	firstNum := first.Migrations["demo/001"].Number

	// Add a second plugin migration; the first must keep its number.
	p.migrations = append(p.migrations, plugins.Migration{
		Version: 2, Description: "second", Up: "ALTER TABLE demo ADD COLUMN name TEXT;", Down: "ALTER TABLE demo DROP COLUMN name;",
	})
	cfg.Plugins = []plugins.Plugin{p}
	if err := NewExporter(cfg).Export(); err != nil {
		t.Fatalf("second export: %v", err)
	}

	second := readManifest(t, dir)
	if got := second.Migrations["demo/001"].Number; got != firstNum {
		t.Fatalf("demo/001 was renumbered: %d -> %d", firstNum, got)
	}
	if got := second.Migrations["demo/002"].Number; got != firstNum+1 {
		t.Fatalf("demo/002 number = %d, want %d", got, firstNum+1)
	}
}

func TestExportAdoptsExistingAegisFiles(t *testing.T) {
	dir := t.TempDir()
	migrations, err := auth.GetMigrations(plugins.DialectPostgres)
	if err != nil {
		t.Fatalf("get auth migrations: %v", err)
	}
	// Simulate the old exporter's flat output (no manifest).
	for i, m := range migrations {
		name := filepath.Join(dir, oldAegisName(i+1, "auth", m.Description))
		if err := os.WriteFile(name, []byte(m.Up), 0600); err != nil {
			t.Fatal(err)
		}
	}

	exp := NewExporter(Config{Format: FormatSQL, Dialect: plugins.DialectPostgres, OutputDir: dir, AuthOnly: true})
	if err := exp.Export(); err != nil {
		t.Fatalf("export: %v", err)
	}

	mf := readManifest(t, dir)
	for i, m := range migrations {
		id := "auth/" + pad3(m.Version)
		entry, ok := mf.Migrations[id]
		if !ok {
			t.Fatalf("manifest missing %s", id)
		}
		if entry.Number != i+1 {
			t.Fatalf("%s number = %d, want adopted %d", id, entry.Number, i+1)
		}
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
	mf := readManifest(t, dir)
	entry, ok := mf.Migrations["email-otp/001"]
	if !ok {
		t.Fatalf("manifest missing email-otp/001: %+v", mf.Migrations)
	}
	if entry.Number != 1 {
		t.Fatalf("adopted number = %d, want 1", entry.Number)
	}
}

func TestExportRejectsChangedMigration(t *testing.T) {
	dir := t.TempDir()
	p := &testPlugin{name: "demo", migrations: []plugins.Migration{
		{Version: 1, Description: "first", Up: "CREATE TABLE demo (id TEXT);", Down: "DROP TABLE demo;"},
	}}
	cfg := Config{Format: FormatSQL, Dialect: plugins.DialectPostgres, OutputDir: dir, Plugins: []plugins.Plugin{p}}
	if err := NewExporter(cfg).Export(); err != nil {
		t.Fatalf("first export: %v", err)
	}

	p.migrations[0].Up = "CREATE TABLE demo (id TEXT, changed BOOLEAN);"
	err := NewExporter(cfg).Export()
	if err == nil || !strings.Contains(err.Error(), "released migrations must be immutable") {
		t.Fatalf("expected immutable-migration error, got: %v", err)
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
	up := filepath.Join(dir, "000001_aegis_auth_initial.up.sql")
	down := filepath.Join(dir, "000001_aegis_auth_initial.down.sql")
	if _, err := os.Stat(up); err != nil {
		t.Fatalf("missing up migration: %v", err)
	}
	if _, err := os.Stat(down); err != nil {
		t.Fatalf("missing down migration: %v", err)
	}
}

func pad3(n int) string {
	s := "000" + itoa(n)
	return s[len(s)-3:]
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

func oldAegisName(number int, source, description string) string {
	s := "00000" + itoa(number)
	return s[len(s)-5:] + "_aegis_" + source + "_" + description + ".sql"
}

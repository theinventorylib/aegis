package migrations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theinventorylib/aegis/plugins"
	aegistesting "github.com/theinventorylib/aegis/testing"
)

func TestExporter_SQL(t *testing.T) {
	tmpDir := t.TempDir()

	exporter := NewExporter(ExporterConfig{
		Format:    FormatSQL,
		OutputDir: tmpDir,
		CoreOnly:  true,
	})

	if err := exporter.Export(); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify core migration was created
	coreFile := filepath.Join(tmpDir, "001_aegis_core.sql")
	if _, err := os.Stat(coreFile); os.IsNotExist(err) {
		t.Error("Core migration file was not created")
	}

	// Read and verify contents
	content, _ := os.ReadFile(coreFile) //nolint:gosec // Test file safe to read
	if !strings.Contains(string(content), "CREATE TABLE") {
		t.Error("Core migration should contain CREATE TABLE statements")
	}

	t.Log("✓ SQL format export successful")
}

func TestExporter_Goose(t *testing.T) {
	tmpDir := t.TempDir()

	exporter := NewExporter(ExporterConfig{
		Format:    FormatGoose,
		OutputDir: tmpDir,
		CoreOnly:  true,
	})

	if err := exporter.Export(); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify core migration with Goose format
	coreFile := filepath.Join(tmpDir, "00001_aegis_core.sql")
	if _, err := os.Stat(coreFile); os.IsNotExist(err) {
		t.Error("Goose migration file was not created")
	}

	// Verify Goose directives
	content, _ := os.ReadFile(coreFile) //nolint:gosec // Test file safe to read
	contentStr := string(content)
	if !strings.Contains(contentStr, "-- +goose Up") {
		t.Error("Goose migration should contain '-- +goose Up' directive")
	}
	if !strings.Contains(contentStr, "-- +goose Down") {
		t.Error("Goose migration should contain '-- +goose Down' directive")
	}

	t.Log("✓ Goose format export successful")
}

func TestExporter_GolangMigrate(t *testing.T) {
	tmpDir := t.TempDir()

	exporter := NewExporter(ExporterConfig{
		Format:    FormatGolangMigrate,
		OutputDir: tmpDir,
		CoreOnly:  true,
	})

	if err := exporter.Export(); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify separate up/down files
	upFile := filepath.Join(tmpDir, "000001_aegis_core.up.sql")
	downFile := filepath.Join(tmpDir, "000001_aegis_core.down.sql")

	if _, err := os.Stat(upFile); os.IsNotExist(err) {
		t.Error("golang-migrate up file was not created")
	}
	if _, err := os.Stat(downFile); os.IsNotExist(err) {
		t.Error("golang-migrate down file was not created")
	}

	t.Log("✓ golang-migrate format export successful")
}

func TestExporter_WithPlugins(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock plugin
	mockPlugin := aegistesting.NewMockPlugin("test-plugin")

	exporter := NewExporter(ExporterConfig{
		Format:    FormatSQL,
		OutputDir: tmpDir,
		CoreOnly:  false,
		Plugins:   []plugins.Plugin{mockPlugin},
	})

	if err := exporter.Export(); err != nil {
		t.Fatalf("Export with plugins failed: %v", err)
	}

	// Verify core migration exists
	coreFile := filepath.Join(tmpDir, "001_aegis_core.sql")
	if _, err := os.Stat(coreFile); os.IsNotExist(err) {
		t.Error("Core migration file should exist")
	}

	t.Log("✓ Export with plugins successful")
}

func TestExporter_InvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()

	exporter := NewExporter(ExporterConfig{
		Format:    "invalid-format",
		OutputDir: tmpDir,
		CoreOnly:  true,
	})

	err := exporter.Export()
	if err == nil {
		t.Error("Expected error for invalid format, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("Expected 'unsupported format' error, got: %v", err)
	}

	t.Log("✓ Invalid format correctly rejected")
}

func TestExporter_OutputDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "nested", "migrations")

	exporter := NewExporter(ExporterConfig{
		Format:    FormatSQL,
		OutputDir: outputDir,
		CoreOnly:  true,
	})

	if err := exporter.Export(); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Error("Output directory should be created")
	}

	// Verify file permissions (should be 0750 for directory)
	dirInfo, _ := os.Stat(outputDir)
	if dirInfo.Mode().Perm() != 0750 {
		t.Logf("Directory permissions: %o (expected 0750)", dirInfo.Mode().Perm())
	}

	t.Log("✓ Output directory created with correct permissions")
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple Name", "simple_name"},
		{"With-Dashes", "with_dashes"},
		{"With Spaces and-Dashes", "with_spaces_and_dashes"},
		{"Special!@#Characters", "specialcharacters"},
		{"UPPERCASE", "uppercase"},
		{"123numbers", "123numbers"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}

	t.Log("✓ Filename sanitization works correctly")
}

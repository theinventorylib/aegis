package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/theinventorylib/aegis/migrations"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/plugins/email"
	"github.com/theinventorylib/aegis/plugins/oauth"
	"github.com/theinventorylib/aegis/plugins/sms"
)

const version = "1.0.0"

// These variables are set by GoReleaser during build
var (
	commit = "none"
	date   = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "export":
		exportCmd()
	case "version", "--version", "-v":
		fmt.Printf("Aegis CLI v%s\n", version)
		if commit != "none" {
			fmt.Printf("  commit: %s\n", commit)
			fmt.Printf("  built:  %s\n", date)
		}
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func exportCmd() {
	exportFlags := flag.NewFlagSet("export", flag.ExitOnError)
	format := exportFlags.String("format", "sql", "Export format: sql, goose, golang-migrate")
	output := exportFlags.String("output", "./aegis-migrations", "Output directory for migrations")
	pluginsStr := exportFlags.String("plugins", "all", "Comma-separated list of plugins (all, or e.g., sms,email,oauth)")
	coreOnly := exportFlags.Bool("core-only", false, "Export only core schema (no plugins)")

	exportFlags.Parse(os.Args[2:])

	// Validate format
	var exportFormat migrations.ExportFormat
	switch strings.ToLower(*format) {
	case "sql":
		exportFormat = migrations.FormatSQL
	case "goose":
		exportFormat = migrations.FormatGoose
	case "golang-migrate":
		exportFormat = migrations.FormatGolangMigrate
	default:
		fmt.Printf("Error: Invalid format '%s'. Must be one of: sql, goose, golang-migrate\n", *format)
		os.Exit(1)
	}

	// Get plugins
	var pluginList []plugins.Plugin
	if !*coreOnly {
		pluginList = getPlugins(*pluginsStr)
		if pluginList == nil {
			fmt.Printf("Error: Invalid plugin selection '%s'\n", *pluginsStr)
			os.Exit(1)
		}
	}

	// Create exporter
	exporter := migrations.NewExporter(migrations.ExporterConfig{
		Format:    exportFormat,
		OutputDir: *output,
		CoreOnly:  *coreOnly,
		Plugins:   pluginList,
	})

	// Export
	fmt.Printf("Exporting Aegis migrations...\n")
	fmt.Printf("  Format: %s\n", exportFormat)
	fmt.Printf("  Output: %s\n", *output)
	if *coreOnly {
		fmt.Printf("  Scope: Core only\n")
	} else {
		pluginNames := make([]string, len(pluginList))
		for i, p := range pluginList {
			pluginNames[i] = p.Name()
		}
		fmt.Printf("  Plugins: %s\n", strings.Join(pluginNames, ", "))
	}

	if err := exporter.Export(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Successfully exported migrations to %s\n", *output)
}

// getPlugins returns plugin instances based on the selection string
func getPlugins(selection string) []plugins.Plugin {
	// For migration export, we don't need full DB initialization
	// We only need to call GetMigrations() which doesn't require DB connection
	availablePlugins := map[string]plugins.Plugin{
		"email": email.New(&email.Config{}),
		"sms":   sms.New(&sms.Config{}),
		"oauth": oauth.New(&oauth.Config{}),
	}

	if selection == "all" {
		result := make([]plugins.Plugin, 0, len(availablePlugins))
		// Return in a consistent order
		order := []string{"email", "sms", "oauth"}
		for _, name := range order {
			if p, ok := availablePlugins[name]; ok {
				result = append(result, p)
			}
		}
		return result
	}

	// Parse comma-separated list
	names := strings.Split(selection, ",")
	result := make([]plugins.Plugin, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if p, ok := availablePlugins[name]; ok {
			result = append(result, p)
		} else {
			fmt.Printf("Warning: Unknown plugin '%s' (available: email, sms, oauth)\n", name)
			return nil
		}
	}

	return result
}

func printUsage() {
	fmt.Println("Aegis CLI - Migration Export Tool")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  aegis export [flags]")
	fmt.Println()
	fmt.Println("FLAGS:")
	fmt.Println("  --format string       Export format (default: sql)")
	fmt.Println("                        Options: sql, goose, golang-migrate")
	fmt.Println("  --output string       Output directory (default: ./aegis-migrations)")
	fmt.Println("  --plugins string      Plugins to include (default: all)")
	fmt.Println("                        Examples: all, sms,email, oauth")
	fmt.Println("  --core-only          Export only core schema")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  # Export all migrations as Goose format")
	fmt.Println("  aegis export --format goose --output ./migrations")
	fmt.Println()
	fmt.Println("  # Export only email and SMS plugins as SQL")
	fmt.Println("  aegis export --format sql --plugins sms,email")
	fmt.Println()
	fmt.Println("  # Export only core schema")
	fmt.Println("  aegis export --core-only --output ./migrations/core")
	fmt.Println()
	fmt.Println("AVAILABLE PLUGINS:")
	fmt.Println("  - email          Email verification and authentication")
	fmt.Println("  - sms            SMS/phone number verification")
	fmt.Println("  - oauth          OAuth provider integrations")
	fmt.Println()
	fmt.Println("OTHER COMMANDS:")
	fmt.Println("  aegis version    Show version information")
	fmt.Println("  aegis help       Show this help message")
}

// Dummy context to initialize plugins (they won't actually connect to DB)
var _ = context.Background()

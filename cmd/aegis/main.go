// Package main provides the Aegis CLI tool for database migration management.
//
// The Aegis CLI exports database migration files for different migration tools:
//   - Raw SQL: Plain SQL files for manual execution
//   - Goose: Migration files for pressly/goose
//   - golang-migrate: Migration files for golang-migrate/migrate
//
// Features:
//   - Multi-database support: PostgreSQL, MySQL, SQLite
//   - Plugin selection: Export specific plugins or all plugins
//   - Multiple formats: SQL, Goose, golang-migrate
//   - Core auth: Always includes base authentication tables
//
// Commands:
//   - aegis export: Export migration files
//   - aegis version: Show version information
//   - aegis help: Show usage information
//
// Quick Start:
//
//	# Export auth schema for PostgreSQL
//	aegis export --dialect postgres --output ./migrations
//
//	# Export auth + all plugins as Goose format
//	aegis export --dialect postgres --format goose --plugins all
//
//	# Export auth + specific plugins
//	aegis export --dialect postgres --plugins oauth,jwt,organizations
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/theinventorylib/aegis/exporter"
	iversion "github.com/theinventorylib/aegis/internal/version"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/plugins/admin"
	"github.com/theinventorylib/aegis/plugins/emailotp"
	"github.com/theinventorylib/aegis/plugins/jwt"
	"github.com/theinventorylib/aegis/plugins/oauth"
	"github.com/theinventorylib/aegis/plugins/openapi"
	"github.com/theinventorylib/aegis/plugins/organizations"
	"github.com/theinventorylib/aegis/plugins/sms"
)

// version is injected by goreleaser at build time via -X internal/version.Version; falls back to build info or "dev".
var version = iversion.Version

// commit and date are set by GoReleaser during build for release tracking
var (
	commit = "none"    // Git commit hash
	date   = "unknown" // Build timestamp
)

// main is the CLI entry point.
//
// Parses command-line arguments and dispatches to the appropriate command handler.
// Exits with code 1 on errors, code 0 on success.
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

// exportCmd handles the 'aegis export' command.
//
// This command exports database migration files for Aegis authentication tables.
// It supports:
//   - Multiple database dialects (PostgreSQL, MySQL, SQLite)
//   - Multiple migration formats (SQL, Goose, golang-migrate)
//   - Plugin selection (all, specific plugins, or auth-only)
//
// Flags:
//
//	--dialect: Database dialect (postgres, mysql, sqlite) - REQUIRED
//	--format: Migration format (sql, goose, golang-migrate) - Default: sql
//	--output: Output directory - Default: ./migrations
//	--plugins: Comma-separated plugin list or "all" - Default: none (auth only)
//	--auth-only: Export only auth schema, no plugins - Deprecated, use --plugins=""
//
// Examples:
//
//	aegis export --dialect postgres
//	aegis export --dialect postgres --plugins all
//	aegis export --dialect mysql --plugins oauth,jwt
//	aegis export --dialect postgres --format goose --plugins all
func exportCmd() {
	exportFlags := flag.NewFlagSet("export", flag.ExitOnError)
	format := exportFlags.String("format", "sql", "Export format: sql, goose, golang-migrate")
	dialect := exportFlags.String("dialect", "", "Database dialect: postgres, mysql, sqlite (required)")
	output := exportFlags.String("output", "./migrations", "Output directory for migrations")
	pluginsStr := exportFlags.String("plugins", "", "Comma-separated list of plugins (all, or e.g., emailotp,sms,oauth)")
	authOnly := exportFlags.Bool("auth-only", false, "Export only auth schema (no plugins)")

	if err := exportFlags.Parse(os.Args[2:]); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Validate format
	var expFormat exporter.ExportFormat
	switch strings.ToLower(*format) {
	case "sql":
		expFormat = exporter.ExportFormat("sql")
	case "goose":
		expFormat = exporter.ExportFormat("goose")
	case "golang-migrate":
		expFormat = exporter.ExportFormat("golang-migrate")
	default:
		fmt.Printf("Error: Invalid format '%s'. Must be one of: sql, goose, golang-migrate\n", *format)
		os.Exit(1)
	}

	// Require dialect flag (no autodetect).
	if strings.TrimSpace(*dialect) == "" {
		fmt.Println("Error: --dialect is required. Must be one of: postgres, mysql, sqlite")
		printUsage()
		os.Exit(1)
	}

	var dialectStr string
	switch strings.ToLower(strings.TrimSpace(*dialect)) {
	case "postgres", "postgresql":
		dialectStr = "postgres"
	case "mysql":
		dialectStr = "mysql"
	case "sqlite", "sqlite3":
		dialectStr = "sqlite"
	default:
		fmt.Printf("Error: invalid dialect '%s'. Must be one of: postgres, mysql, sqlite\n", *dialect)
		os.Exit(1)
	}

	// Parse plugin names
	var pluginNames []string
	if *pluginsStr != "" {
		for _, name := range strings.Split(*pluginsStr, ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				pluginNames = append(pluginNames, name)
			}
		}
	}

	// Get plugin instances for export
	// Only export plugins if explicitly requested (not auth-only and plugins specified)
	var pluginList []plugins.Plugin
	if !*authOnly && len(pluginNames) > 0 {
		pluginList = getPluginsForExport(plugins.Dialect(dialectStr), pluginNames)
	}

	// Create exporter
	exporterObj := exporter.NewExporter(exporter.Config{
		Format:    expFormat,
		Dialect:   plugins.Dialect(dialectStr),
		OutputDir: *output,
		AuthOnly:  *authOnly,
		Plugins:   pluginList,
	})

	// Export
	fmt.Printf("Exporting Aegis migrations...\n")
	fmt.Printf("  Format: %s\n", *format)
	fmt.Printf("  Dialect: %s\n", dialectStr)
	fmt.Printf("  Output: %s\n", *output)
	if len(pluginList) > 0 {
		if len(pluginNames) == 1 && pluginNames[0] == "all" {
			fmt.Printf("  Scope: Auth + All plugins\n")
		} else {
			fmt.Printf("  Scope: Auth + Plugins [%s]\n", strings.Join(pluginNames, ", "))
		}
	} else {
		fmt.Printf("  Scope: Auth only\n")
	}

	if err := exporterObj.Export(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Successfully exported migrations to %s\n", *output)
	fmt.Printf("  (existing _aegis_ files were replaced; your own files were not modified)\n")
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
	fmt.Println("  --output string       Path to your migrations folder (default: ./migrations)")
	fmt.Println("                        Existing files whose names contain \"_aegis_\" are")
	fmt.Println("                        replaced; your own migration files are never touched.")
	fmt.Println("  --dialect string      Database dialect to use (required: postgres, mysql, sqlite)")
	fmt.Println("  --plugins string      Plugins to include (default: none, auth only)")
	fmt.Println("                        Examples: all, emailotp,sms, oauth,jwt,admin")
	fmt.Println("  --auth-only          Export only auth schema (alias for no plugins, deprecated)")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  # Export auth only into an existing folder (aegis files replaced, yours kept)")
	fmt.Println("  aegis export --dialect postgres --output ./migrations")
	fmt.Println()
	fmt.Println("  # Export auth + all plugins as Goose format")
	fmt.Println("  aegis export --dialect postgres --format goose --plugins all")
	fmt.Println()
	fmt.Println("  # Export auth + specific plugins as SQL")
	fmt.Println("  aegis export --dialect postgres --plugins emailotp,sms")
	fmt.Println()
	fmt.Println("  # Export auth + JWT and OAuth plugins for MySQL")
	fmt.Println("  aegis export --dialect mysql --plugins jwt,oauth --output ./migrations")
	fmt.Println()
	fmt.Println("  # Export auth + all plugins as golang-migrate format")
	fmt.Println("  aegis export --dialect postgres --format golang-migrate --plugins all")
	fmt.Println()
	fmt.Println("NAMING CONVENTION:")
	fmt.Println("  Aegis-generated files always contain \"_aegis_\" in their name")
	fmt.Println("  (e.g. 00001_aegis_auth_initial.sql). Do not use that substring in")
	fmt.Println("  your own migration filenames — those files will be overwritten on")
	fmt.Println("  the next export.")
	fmt.Println()
	fmt.Println("AVAILABLE PLUGINS:")
	fmt.Println("  - admin          Administrative endpoints for user management")
	fmt.Println("  - emailotp       Email verification and authentication")
	fmt.Println("  - jwt            JWT token generation and validation")
	fmt.Println("  - oauth          OAuth provider integrations (Google, GitHub, etc.)")
	fmt.Println("  - openapi        OpenAPI 3.0 documentation with Scalar UI")
	fmt.Println("  - organizations  Multi-tenant organization support")
	fmt.Println("  - sms            SMS/phone number verification")
	fmt.Println()
	fmt.Println("OTHER COMMANDS:")
	fmt.Println("  aegis version    Show version information")
	fmt.Println("  aegis help       Show this help message")
}

// Dummy context to keep some plugin initializers compatible (no DB connection)
var _ = context.Background()

// getPluginsForExport creates plugin instances for migration export.
//
// These plugin instances are lightweight and don't require:
//   - Database connections (nil DB)
//   - Functional stores (nil stores)
//   - Runtime configuration (only migrations/schemas needed)
//
// This function is used by the export command to get migration files from plugins
// without initializing the full authentication system.
//
// Plugin Selection:
//   - Empty list or ["all"]: Returns all available plugins
//   - Specific plugins: Returns only the named plugins
//   - Unknown plugins: Prints warning and skips
//
// Available plugins:
//   - admin: Administrative user management
//   - emailotp: Email OTP verification
//   - jwt: JWT token authentication
//   - oauth: OAuth provider integration
//   - openapi: API documentation (no migrations)
//   - organizations: Multi-tenant organizations
//   - sms: SMS OTP verification
//
// Parameters:
//   - dialect: Database dialect for migration generation
//   - pluginNames: List of plugin names to export, or ["all"] for all plugins
//
// Returns:
//   - []plugins.Plugin: Plugin instances for migration export
func getPluginsForExport(dialect plugins.Dialect, pluginNames []string) []plugins.Plugin {
	// Create a map of all available plugins
	allPlugins := map[string]plugins.Plugin{
		"admin":         admin.New(nil, dialect),
		"emailotp":      emailotp.New(nil, nil, dialect),
		"jwt":           jwt.New(nil, nil, dialect),
		"oauth":         oauth.New(nil, nil, dialect),
		"openapi":       openapi.New(nil),
		"organizations": organizations.New(nil, dialect),
		"sms":           sms.New(nil, nil, dialect),
	}

	// If "all" or empty list, return all plugins
	if len(pluginNames) == 0 || (len(pluginNames) == 1 && pluginNames[0] == "all") {
		result := make([]plugins.Plugin, 0, len(allPlugins))
		for _, p := range allPlugins {
			result = append(result, p)
		}
		return result
	}

	// Return selected plugins (preallocate capacity to avoid reallocs)
	result := make([]plugins.Plugin, 0, len(pluginNames))
	for _, name := range pluginNames {
		if p, ok := allPlugins[name]; ok {
			result = append(result, p)
		} else {
			fmt.Printf("Warning: unknown plugin '%s', skipping\n", name)
		}
	}

	return result
}

# Aegis Migrations

## Overview
Aegis **does not auto-create tables**. You are responsible for running your own migrations using your preferred migration tool.

**Recommended Approach:** Use the Aegis CLI to export migrations in your preferred format, then run them with your existing migration workflow.

## Quick Start with CLI (Recommended)

### 1. Install Aegis CLI

```bash
go install github.com/theinventorylib/aegis/cmd/aegis@latest
```

### 2. Export Migrations

```bash
# Export all migrations (core + all plugins) in Goose format
aegis export --format goose --output ./migrations/aegis

# Export only specific plugins
aegis export --format sql --plugins email,sms --output ./migrations/aegis

# Export only core schema
aegis export --core-only --output ./migrations/aegis
```

### 3. Run with Your Migration Tool

**Goose:**
```bash
goose -dir ./migrations/aegis postgres "your_connection_string" up
```

**golang-migrate:**
```bash
aegis export --format golang-migrate --output ./migrations/aegis
migrate -path ./migrations/aegis -database "postgres://..." up
```

**Raw SQL:**
```bash
aegis export --format sql --output ./migrations/aegis
psql your_database < ./migrations/aegis/001_aegis_core.sql
psql your_database < ./migrations/aegis/002_aegis_email_001.sql
# etc.
```

## CLI Reference

### Export Command

```bash
aegis export [flags]
```

**Flags:**
- `--format string` - Export format: `sql`, `goose`, `golang-migrate` (default: `sql`)
- `--output string` - Output directory (default: `./aegis-migrations`)
- `--plugins string` - Comma-separated plugin list or `all` (default: `all`)
- `--core-only` - Export only core schema (no plugins)

**Available Plugins:**
- `email` - Email verification
- `sms` - SMS/phone verification
- `oauth` - OAuth providers

**Examples:**

```bash
# Export everything as Goose format
aegis export --format goose --output ./db/migrations

# Export only email and SMS plugins as SQL
aegis export --format sql --plugins email,sms

# Export core only for initial setup
aegis export --core-only

# Export for golang-migrate
aegis export --format golang-migrate --plugins oauth,email
```

## Migration Tools

### 1. Raw SQL (psql)
```bash
# Core schema (required)
psql your_database < migrations/schema.sql
```

### 2. sqlc
```bash
cd migrations/sqlc
sqlc generate
```

Configuration in `migrations/sqlc/sqlc.yaml`

### 3. Goose
```bash
# Run core migration
goose -dir migrations/goose postgres "your_connection_string" up
```

Core migration: `migrations/goose/00001_aegis_core.sql`

### 4. golang-migrate
```bash
migrate -path migrations/golang-migrate -database "postgres://..." up
```

## Plugin Migrations

Aegis uses a **plugin architecture** where:
- **Core schema** contains only 5 tables: `user`, `accounts`, `verification`, `session`, `jwks`
- **Plugins** manage their own schemas: `plugins_sms`, `plugins_email`, `plugins_oauth`

### Accessing Plugin SQL

Plugins provide their migrations programmatically via `GetMigrations()`:

```go
import "github.com/theinventorylib/aegis/plugins/sms"

// Get plugin instance
smsPlugin := sms.New(&sms.Config{DBPool: pool})

// Get migrations
migrations := smsPlugin.GetMigrations()
for _, m := range migrations {
    fmt.Println("Version:", m.Version)
    fmt.Println("SQL Up:", m.Up)
    fmt.Println("SQL Down:", m.Down)
}
```

### Running Plugin Migrations

**Option 1: Extract and run SQL manually**
```go
// Get the SQL
smsPlugin := sms.New(&sms.Config{DBPool: pool})
migrations := smsPlugin.GetMigrations()

// Run with psql
for _, m := range migrations {
    // Write m.Up to a file and run it
    os.WriteFile("plugin_migration.sql", []byte(m.Up), 0644)
    exec.Command("psql", "your_db", "-f", "plugin_migration.sql").Run()
}
```

**Option 2: Run programmatically**
```go
// Execute directly in your migration code
for _, m := range migrations {
    _, err := db.Exec(context.Background(), m.Up)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Migration Ordering

The Aegis exporter generates migrations with the following numbering strategy:

**Core Migration**: Always numbered `001` (or `00001` for Goose format)

**Plugin Migrations**: Numbered starting from `002` onwards:
- Each plugin gets a block of 100 migration numbers
- Plugin 1: `002-101`
- Plugin 2: `102-201`
- Plugin 3: `202-301`
- etc.

**Example with 3 plugins**:
```
001_aegis_core.sql              # Core schema (required)
002_aegis_email_001.sql         # Email plugin migration 1
102_aegis_sms_001.sql           # SMS plugin migration 1
202_aegis_oauth_001.sql         # OAuth plugin migration 1
```

**Plugin Dependencies**:
- Plugins declare their table dependencies via `RequiresTables()`
- The exporter does **NOT** automatically reorder migrations based on dependencies
- You are responsible for ensuring plugins are exported/run in the correct order
- Always run **core migrations first** before any plugin migrations

**Example of plugin dependencies**:
```go
// Email plugin requires core tables
func (p *Plugin) RequiresTables() []string {
    return []string{"auth.user", "auth.accounts"}
}
```

**Recommended Order**:
1. Core schema (`auth.user`, `auth.accounts`, `auth.verification`, `auth.session`)
2. Foundation plugins (email, SMS, OAuth)
3. Higher-level plugins (admin, organizations)


## Core Tables

Aegis requires these **5 core tables**:
- `auth.user` - User identities
- `auth.accounts` - Authentication accounts (password, OAuth, etc.)
- `auth.verification` - Generic verification tokens
- `auth.session` - User sessions
- `auth.jwks` - JSON Web Key Sets

## Plugin Tables

### SMS Plugin (`plugins_sms`)
- `plugins_sms.verifications` - Phone number verifications

### Email Plugin (`plugins_email`)
- `plugins_email.verifications` - Email verifications (OTP and token-based)

### OAuth Plugin (`plugins_oauth`)
- `plugins_oauth.connections` - OAuth provider connections

## Notes

- The `auth` schema and `update_updated_at_column()` function are required
- All templates use the same core schema structure
- Choose the migration tool that fits your workflow
- **Plugins are optional** - only run migrations for plugins you use
- Custom fields can be added via JSONB `metadata` columns

# Aegis CLI

Command-line tool for exporting Aegis database migrations.

## Installation

```bash
go install github.com/theinventorylib/aegis/cmd/aegis@latest
```

## Usage

### Export Command

```bash
aegis export [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | string | `sql` | Export format: `sql`, `goose`, `golang-migrate` |
| `--output` | string | `./aegis-migrations` | Output directory for migrations |
| `--plugins` | string | `all` | Comma-separated plugin list or `all` |
| `--core-only` | bool | `false` | Export only core schema (no plugins) |

### Available Plugins

- `email` - Email verification and authentication
- `sms` - SMS/phone number verification
- `oauth` - OAuth provider integrations

## Examples

### Export All Migrations (Core + All Plugins)

```bash
aegis export --format goose --output ./migrations/aegis
```

This exports migrations for the core schema and all available plugins in Goose format.

### Export Specific Plugins

```bash
aegis export --format sql --plugins email,sms --output ./migrations
```

Exports only core schema plus email and SMS plugin migrations.

### Export Core Schema Only

```bash
aegis export --core-only --output ./migrations/core
```

Exports only the core Aegis schema (5 tables) without any plugins.

### Export for golang-migrate

```bash
aegis export --format golang-migrate --output ./db/migrations
```

Exports migrations in golang-migrate format (separate `.up.sql` and `.down.sql` files).

## Migration Formats

### SQL Format

Plain SQL files with numeric prefixes:

```
001_aegis_core.sql
002_aegis_email_001.sql
003_aegis_sms_001.sql
```

### Goose Format

Goose-compatible files with `-- +goose Up/Down` markers:

```
00001_aegis_core.sql
00002_aegis_email_add_email_columns_to_authuser_table.sql
00003_aegis_sms_add_phone_number_column_to_authuser_table.sql
```

### golang-migrate Format

Separate up and down migration files:

```
000001_aegis_core.up.sql
000001_aegis_core.down.sql
000002_aegis_email_add_email_columns.up.sql
000002_aegis_email_add_email_columns.down.sql
```

## Integration with Migration Tools

### Goose

```bash
# Export migrations
aegis export --format goose --output ./migrations

# Run migrations
goose -dir ./migrations postgres "your_connection_string" up
```

### golang-migrate

```bash
# Export migrations
aegis export --format golang-migrate --output ./migrations

# Run migrations
migrate -path ./migrations -database "postgres://..." up
```

### Raw SQL

```bash
# Export migrations
aegis export --format sql --output ./migrations

# Run with psql
for f in migrations/*.sql; do
    psql $DATABASE_URL < $f
done
```

## How It Works

1. **Plugin Discovery**: CLI instantiates requested plugins (without DB connection)
2. **Migration Extraction**: Calls `GetMigrations()` on each plugin
3. **Format Conversion**: Converts to target format (SQL, Goose, or golang-migrate)
4. **File Generation**: Writes properly ordered migration files

The CLI ensures:
- ✅ Core schema is always exported first
- ✅ Plugins are exported in consistent order
- ✅ Both up and down migrations are included
- ✅ File naming follows format conventions

## Other Commands

### Version

```bash
aegis version
```

Shows the CLI version.

### Help

```bash
aegis help
```

Displays usage information.

## Notes

- The CLI does **not** require a database connection
- Plugins are instantiated only to extract their migration SQL
- Output files can be committed to version control
- Re-running the export overwrites existing files

## Troubleshooting

### "Unknown plugin" error

Make sure you're using valid plugin names: `email`, `sms`, `oauth`

```bash
# ✅ Correct
aegis export --plugins email,sms

# ❌ Incorrect
aegis export --plugins emails,text-message
```

### Migrations not in expected order

The CLI ensures proper ordering:
- Core: `001` (SQL), `00001` (Goose), `000001` (golang-migrate)
- Plugins: Sequential numbering after core

If you need custom ordering, use raw SQL format and rename files manually.

## See Also

- [Main README](../../README.md) - Project overview
- [Migration Guide](../../migrations/README.md) - Migration strategies
- [Plugin Development](../../docs/custom-plugins.md) - Creating custom plugins

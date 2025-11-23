# CLI Reference

The Aegis CLI tool is primarily used for exporting database migrations. Since Aegis does not automatically migrate your database at runtime, you use the CLI to generate the necessary SQL files.

## Installation

```bash
go install github.com/theinventorylib/aegis/cmd/aegis@latest
```

## Commands

### `export`

Exports database migrations to a specified directory.

**Usage:**

```bash
aegis export [flags]
```

**Flags:**

- `--format`: The format of the migrations. Supported values: `sql` (default), `goose`, `golang-migrate`.
- `--output`: The output directory. Default: `./aegis-migrations`.
- `--plugins`: Comma-separated list of plugins to include migrations for. Default: `all`. Use `core` to export only the core schema.
- `--core-only`: (Deprecated, use `--plugins=core`) Export only the core schema.

**Examples:**

```bash
# Export everything in standard SQL format
aegis export --output ./migrations

# Export for Goose
aegis export --format goose --output ./migrations/goose

# Export only core and email plugin
aegis export --plugins core,email --output ./migrations
```

## Migration Strategy

1. **Export**: Run `aegis export` to generate migration files.
2. **Review**: Check the generated files in your output directory.
3. **Apply**: Use your existing migration tool (like Goose, golang-migrate, or just `psql`) to apply the migrations to your database.

This approach ensures that **you** are always in control of your database schema changes.

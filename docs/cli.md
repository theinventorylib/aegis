# CLI Reference

The Aegis CLI tool exports database migrations for the core schema and plugins. Since Aegis does not automatically migrate your database at runtime, you use the CLI to generate migration files that you control and apply using your preferred migration tool.

## Installation

```bash
go install github.com/theinventorylib/aegis/cmd/aegis@latest
```

Verify installation:

```bash
aegis version
```

---

## Commands

### `export`

Exports database migrations to a specified directory.

**Usage:**

```bash
aegis export [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | string | `sql` | Export format: `sql`, `goose`, `golang-migrate` |
| `--output` | string | `./aegis-migrations` | Output directory for migrations |
| `--plugins` | string | `all` | Comma-separated plugin list or `all` |
| `--core-only` | bool | `false` | Export only core schema (no plugins) |

---

## Available Plugins

The CLI supports exporting migrations for the following plugins:

| Plugin | Description |
|--------|-------------|
| `email` | Email verification and authentication |
| `sms` | SMS/phone number verification |
| `oauth` | OAuth provider integrations (Google, GitHub, etc.) |
| `jwt` | JWT token generation and validation |
| `bearer` | Bearer token authentication support |
| `openapi` | OpenAPI 3.0 documentation with Scalar UI |
| `admin` | Administrative endpoints for user management |
| `organizations` | Multi-tenant organization support |

---

## Export Formats

### SQL Format

Plain SQL files with numeric prefixes:

```
001_aegis_core.sql
002_aegis_password_001.sql
003_aegis_email_001.sql
004_aegis_sms_001.sql
```

**Usage:**
```bash
aegis export --format sql --output ./migrations
```

**Apply with psql:**
```bash
for f in migrations/*.sql; do
    psql $DATABASE_URL < $f
done
```

### Goose Format

Goose-compatible files with `-- +goose Up/Down` markers:

```
00001_aegis_core.sql
00002_aegis_password_add_password_hash.sql
00003_aegis_email_add_email_columns.sql
```

**Usage:**
```bash
aegis export --format goose --output ./migrations
```

**Apply with Goose:**
```bash
goose -dir ./migrations postgres "$DATABASE_URL" up
```

### golang-migrate Format

Separate up and down migration files:

```
000001_aegis_core.up.sql
000001_aegis_core.down.sql
000002_aegis_password.up.sql
000002_aegis_password.down.sql
```

**Usage:**
```bash
aegis export --format golang-migrate --output ./migrations
```

**Apply with golang-migrate:**
```bash
migrate -path ./migrations -database "$DATABASE_URL" up
```

---

## Examples

### Export All Migrations (Core + All Plugins)

```bash
aegis export --format goose --output ./migrations/aegis
```

This exports migrations for the core schema and all 8 available plugins in Goose format.

### Export Core Schema Only

```bash
aegis export --core-only --output ./migrations/core
```

Exports only the 4 core tables (auth.user, auth.accounts, auth.session, auth.verification).

### Export Specific Plugins

```bash
# Core + password + email
aegis export --plugins password,email --output ./migrations

# Core + all auth plugins
aegis export --plugins password,email,sms,oauth --output ./migrations

# Core + JWT + admin + openapi
aegis export --plugins jwt,admin,openapi --output ./migrations
```

### Export for Different Databases

```bash
# PostgreSQL with Goose
aegis export --format goose --output ./migrations/postgres

# MySQL with golang-migrate
aegis export --format golang-migrate --output ./migrations/mysql

# SQLite with raw SQL
aegis export --format sql --output ./migrations/sqlite
```

---

## Migration Strategy

### Recommended Workflow

1. **Export Migrations**
   ```bash
   aegis export --format goose --plugins all --output ./migrations
   ```

2. **Review Generated Files**
   ```bash
   ls -la ./migrations
   cat ./migrations/00001_aegis_core.sql
   ```

3. **Commit to Version Control**
   ```bash
   git add migrations/
   git commit -m "Add Aegis migrations"
   ```

4. **Apply to Database**
   ```bash
   # Development
   goose -dir ./migrations postgres "$DEV_DATABASE_URL" up
   
   # Production
   goose -dir ./migrations postgres "$PROD_DATABASE_URL" up
   ```

### Integration with Migration Tools

#### Goose

```bash
# Install Goose
go install github.com/pressly/goose/v3/cmd/goose@latest

# Export migrations
aegis export --format goose --output ./migrations

# Apply migrations
goose -dir ./migrations postgres "$DATABASE_URL" up

# Rollback
goose -dir ./migrations postgres "$DATABASE_URL" down

# Check status
goose -dir ./migrations postgres "$DATABASE_URL" status
```

#### golang-migrate

```bash
# Install golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Export migrations
aegis export --format golang-migrate --output ./migrations

# Apply migrations
migrate -path ./migrations -database "$DATABASE_URL" up

# Rollback
migrate -path ./migrations -database "$DATABASE_URL" down 1

# Force version (if stuck)
migrate -path ./migrations -database "$DATABASE_URL" force 1
```

#### Raw SQL

```bash
# Export migrations
aegis export --format sql --output ./migrations

# Apply with psql (PostgreSQL)
for f in migrations/*.sql; do
    psql $DATABASE_URL < $f
done

# Apply with mysql (MySQL)
for f in migrations/*.sql; do
    mysql -h localhost -u user -p database < $f
done
```

---

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

---

## Troubleshooting

### "Unknown plugin" error

Make sure you're using valid plugin names:

```bash
# ✅ Correct
aegis export --plugins password,email,sms

# ❌ Incorrect
aegis export --plugins passwords,emails,text-message
```

**Valid plugin names**: `email`, `sms`, `oauth`, `jwt`, `bearer`, `openapi`, `admin`, `organizations`

### Migrations not in expected order

The CLI ensures proper ordering:
- Core: `001` (SQL), `00001` (Goose), `000001` (golang-migrate)
- Plugins: Sequential numbering after core

If you need custom ordering, use raw SQL format and rename files manually.

### Output directory already exists

The CLI will overwrite existing files. To preserve old migrations:

```bash
# Backup existing migrations
mv ./migrations ./migrations.backup

# Export new migrations
aegis export --output ./migrations
```

### Missing migrations for a plugin

Ensure the plugin is specified:

```bash
# Check which plugins are included
aegis export --plugins password,email --output ./migrations

# Or export all plugins
aegis export --plugins all --output ./migrations
```

---

## Best Practices

1. **Version Control**: Always commit migrations to git
2. **Review Before Apply**: Check generated SQL before running
3. **Test First**: Apply to development database before production
4. **Backup**: Backup production database before migrations
5. **Use Migration Tools**: Use Goose or golang-migrate for rollback support

---

## See Also

- [Database Setup Guide](./database-setup.md) - Advanced database configuration
- [Getting Started](./getting-started.md) - Basic Aegis setup
- [Plugins](./plugins.md) - Plugin documentation

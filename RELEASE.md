# Release Process

Aegis is a Go package (library) used as a dependency in Go projects.

## Installation

Users install Aegis as a Go module dependency:

```bash
go get github.com/theinventorylib/aegis@latest
```

## Version Management

**Semantic Versioning** via Git tags:
- `v1.0.0` - Major release
- `v1.1.0` - Minor release (new features, backward compatible)
- `v1.0.1` - Patch release (bug fixes)
- `v2.0.0` - Breaking changes

## Creating a Release

1. **Update version references** (if any in documentation)

2. **Create and push a tag:**
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

3. **Users can then install:**
   ```bash
   go get github.com/theinventorylib/aegis@v1.0.0
   ```

## Go Module Proxy

Releases are automatically available through:
- Go module proxy (proxy.golang.org)
- Direct from GitHub
- pkg.go.dev for documentation

## Notes

- The package is meant to be imported, not run as a binary
- Users access migrations from the `migrations/` directory
- Plugin migrations are in `plugins/*/migrations/`
- Breaking API changes require major version bump
- GitHub Releases can include migration examples and changelogs

## Migration Distribution

Since users can't easily access migration files from the Go module, they should:

1. **Copy from GitHub repository:**
   ```bash
   curl -O https://raw.githubusercontent.com/theinventorylib/aegis/main/migrations/schema.sql
   ```

2. **Or use initialization script in their project:**
   ```go
   // Copy migrations to local directory on first run
   ```

3. **Or provide migrations in repository examples**

## Tagging Best Practices

- Tag only stable, tested releases
- Include changelog in GitHub releases
- Document breaking changes clearly
- Use pre-release tags for testing: `v1.0.0-beta.1`

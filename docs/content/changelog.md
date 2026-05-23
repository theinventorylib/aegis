The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.6.1] - 2026-05-23
- chore: release changelog update (d01fca2)
- chore: format core test files (ff8ed4c)
- chore: format and docs deps updates (2d969e4)
- fix: apply Go naming conventions and simplified chi group routing (42553f1)
- Add comprehensive tests for HTTP routing and pagination handling (188c251)


## [Unreleased]

### Added
- Initial documentation site
- Core authentication features
- 8 official plugins (Email OTP, SMS, OAuth, JWT, Bearer, Admin, Organizations, OpenAPI)
- CLI tool for migration export
- **SMS plugin** (`plugins/sms`): phone+password authentication and SMS OTP verification with a pluggable `Provider` interface (Twilio, AWS SNS, Vonage, etc.). Routes: `POST /sms/send`, `POST /sms/verify`, `POST /sms/login`, `POST /sms/register`
- **OAuth token refresh**: `POST /auth/oauth/:provider/refresh` endpoint and `Plugin.RefreshConnection(ctx, userID, provider)` Go API for proactively refreshing provider access tokens
- **Session pagination**: `SessionService.GetUserSessions(ctx, userID, offset, limit)` and `SessionService.CountUserSessions(ctx, userID)` for paginated device management UIs
- **Plugin lifecycle interfaces**: `PluginShutdown` (graceful stop), `PluginRequires` (name-based deps), `PluginVersionRequires` (semver deps), `PluginMinAegisVersion` (framework version gate)
- **`aegis.Shutdown(ctx)`**: graceful shutdown of all registered plugins in reverse priority order
- **`config.WithDialect(d)`**: explicit database dialect selector (`DialectPostgres`, `DialectMySQL`, `DialectSQLite`) — required for MySQL and SQLite deployments
- **MySQL support across all plugins**: auth, admin, emailotp, jwt, organizations, sms, and oauth all ship MySQL-specific sqlc-generated query sets
- **Multi-file default stores**: each plugin's `default_store` is now split into `store.go`, `mysql.go`, `postgres.go`, `sqlite.go`, and `querier.go` for easier maintenance
- **`aegis.Version`**: runtime-accessible framework version string injected by GoReleaser (falls back to build-info `dev`)

### Changed
- Router defaults restructured: `router/routes.go` → `router/defaults/routes.go`; `router/handlers.go` → `router/defaults/handlers.go`
- OpenAPI registration refactored into dedicated `route.go` and `queue.go` files for cleaner per-plugin route metadata registration
- Organizations handlers updated with paginated member and team queries
- Email OTP sender refactored for cleaner template handling
- Admin plugin modularised; admin store split into dialect-specific files matching the new default_store convention

### Deprecated
- N/A

### Removed
- `router/routes.go` and `router/chi.go` (replaced by `router/defaults/`)
- Monolithic `default_store.go` files in auth, admin, emailotp, jwt, and organizations (replaced by per-dialect split files)

### Fixed
- JWT key-rotation now uses the caller-provided context instead of `context.Background()`, preventing context-deadline leaks
- Resolved 68 `golangci-lint` issues (errcheck, revive, dupl)
- Expanded API documentation on all exported types across `auth`, `plugins/oauth/types`, `plugins/emailotp/types`, `plugins/organizations`, and all `default_store` packages

### Security
- All 12 `gosec` findings resolved:
  - **G101** (9 findings): `sqlc`-generated query files excluded from the scan via `-exclude-dir=internal/gen` — SQL constants named `getSessionByToken` are parameterised query templates, not credentials
  - **G124** (3 findings): `core/cookies.go` `http.SetCookie` call-sites annotated `// #nosec G124` — `Secure`, `HttpOnly`, and `SameSite` are always set from caller-supplied config or `CookieOptions`

---

## How to Read This Changelog

- **Added** - New features
- **Changed** - Changes in existing functionality
- **Deprecated** - Soon-to-be removed features
- **Removed** - Now removed features
- **Fixed** - Bug fixes
- **Security** - Vulnerability fixes

---

## Release Notes

Future releases will be documented here. Follow the [GitHub repository](https://github.com/theinventorylib/aegis) for updates.

::callout{icon="i-lucide-bell"}
Subscribe to release notifications on GitHub to stay updated with new features and security patches.
::

## Contributing to Changelog

When submitting PRs, please update this changelog following these guidelines:

1. Add your changes under the `[Unreleased]` section
2. Use the appropriate category (Added, Changed, Fixed, etc.)
3. Write clear, concise descriptions
4. Reference issue/PR numbers when applicable

Example:
```markdown
### Added
- OAuth provider for Microsoft (#123)
- Rate limiting middleware (#124)
```

## Version History

Check the [GitHub Releases](https://github.com/theinventorylib/aegis/releases) page for the complete version history with detailed release notes.

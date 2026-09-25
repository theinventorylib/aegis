The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.1.0] - 2026-09-25

### Added
- **TOTP plugin** (`plugins/totp`): RFC 6238 authenticator-app two-factor authentication with two-phase enrollment, per-session step-up via `RequireVerification`, hashed single-use recovery codes, and `POST /totp/setup|enable|verify|disable|recovery-codes`, `GET /totp/status`.
- **Organization custom roles and per-member permission overrides**: persisted `organization_role` and `member_permission_override` tables, `GET/POST/PUT/DELETE /organizations/:id/roles`, `GET/PUT /organizations/:id/members/:userId/permissions`, and the programmatic `ListRoles` / `CreateRole` / `UpdateRole` / `DeleteRole`, `GetMemberPermissions`, `MemberPermissionOverrides` / `SetMemberPermissionOverrides`.
- **Password reset and email-change flows** in core (`AccountService.RequestPasswordReset` / `ConfirmPasswordReset`, `UserService.RequestEmailChange` / `ConfirmEmailChange`) with email-otp endpoints `POST /email-otp/forgot-password`, `/reset-password`, `/email-change`, `/email-change/confirm`. `AuthConfig` gains `PasswordResetExpiry` / `EmailChangeExpiry` (default 1h).
- **Audit event sinks**: `AuditSink` / `AddAuditSink`, `AuthService.AuditLogger` and `plugins.Aegis.GetAuditLogger`, plus the `user_created`, `user_updated`, `user_deleted` and `email_changed` lifecycle events.
- Optional caller-supplied email-change token: `RequestEmailChange(ctx, userID, newEmail, customToken)`.

### Changed
- **PostgreSQL timestamps are now `TIMESTAMPTZ`** instead of RFC3339 `TEXT`. Apply the new `002_timestamptz` (auth/admin/oauth) and `005_timestamptz` (organizations) migrations; raw SQL that compared timestamp columns as strings must compare timestamps. MySQL and SQLite keep text storage.
- `plugins.Aegis` gained `GetAuditLogger()`; external implementers must add the method.

### Fixed
- TOTP disable and recovery-code regeneration now require a valid code; `RequireVerification` fails closed on credential-lookup errors.
- Organization permission writes cap grants at the actor's own framework permissions and protect the owner's role and overrides; the cap normalizes permission strings so a padded spelling cannot bypass it.
- Timestamptz down-migrations render RFC3339 explicitly instead of a session-dependent `::text` cast; store timestamp parsers log malformed values instead of silently zeroing them.
- `plugins/admin` `001_initial` creates `ban_expiry` as `TIMESTAMPTZ`.
- `VerifyOTPHandler` only marks an address verified for `email_verification` codes.

## [2.0.0] - 2026-09-12

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
- **Permission-based organization/team roles**: `organizations.Config.OrgRoles` / `TeamRoles`, `Permission` constants, `HasOrgPermission` / `HasTeamPermission`, and `RequireOrgPermission` / `RequireTeamPermission` middleware. Custom roles are now first-class; `CanAccessTeam` accepts any role.
- **Username login**: `EmailPasswordHandlers.RegisterWithUsername` and login by email or username (username stored on the credentials account).
- **Email verification gating**: `AuthConfig.RequireEmailVerification`, `AuthService.SetEmailVerificationCheck`, and `POST /auth/email-otp/send-verification` (rate-limited per email).
- **Password policy enforcement** on registration and password change (`AuthService.ValidatePassword`), plus email-format validation and `ErrEmailAlreadyExists` / `ErrUsernameTaken` sentinels.

### Changed
- Router defaults restructured: `router/routes.go` → `router/defaults/routes.go`; `router/handlers.go` → `router/defaults/handlers.go`
- OpenAPI registration refactored into dedicated `route.go` and `queue.go` files for cleaner per-plugin route metadata registration
- Organizations handlers updated with paginated member and team queries
- Email OTP sender refactored for cleaner template handling
- Admin plugin modularised; admin store split into dialect-specific files matching the new default_store convention
- **Upgrade note — master secret**: `config.Validate` now rejects secrets shorter than 32 bytes. Set a ≥32-byte `WithSecret` before upgrading.
- **Upgrade note — JWT refresh keys**: access and refresh keys are now stored separately (`sig` vs `sig-refresh`). `RefreshTokens` falls back to the legacy shared key so outstanding refresh tokens keep working, but tokens minted with the old shared key should be allowed to expire. The JWKS endpoint no longer exposes refresh keys.
- **Upgrade note — password policy**: `AuthConfig.PasswordPolicy` is now enforced on registration and password change; previously it was configured but ignored.

### Deprecated
- N/A in v2.0.0 — the v1.6 compatibility shims that v1.7.0 added (`core/deprecated.go`) and the deprecated organizations `Config.CustomOrgRoles` / `Config.CustomTeamRoles` were removed in this release; the `v1` branch retains them. Removed symbols:
  - Constructors: `NewSessionService`, `NewAccountService`, `NewUserService`, `NewVerificationService`, `NewPluginData`
  - Helpers: `ValidatePassword`, `ValidatePasswordSimple`, `BindAndValidate`, `ValidateMiddleware`, `WrapError`, `IsValidationError`, `MustGetUser`, `MustGetEnrichedUser`, `IsContextInitialized`, `AegisContext`
  - Sanitizers/utilities: `SanitizeFilename`, `SanitizeHTML`, `SanitizeSQL`, `SanitizeSQLIdentifier`, `StripTags`, `NormalizeWhitespace`, `RedactForLog`, `HashShort`, `HashTokenHex`, `IsHashedToken`, `BoolPtr`
  - Types/config: `AccountModel`, `VerificationModel`, `IDGeneratorFunc`, `AuthRateLimitConfig`, `DefaultPasswordHasherConfig`, `DefaultPasswordPolicyConfig`, `GetAuthConfig`, `LoggerAuditLogger`, `NewLoggerAuditLogger`
  - Organizations `Config.CustomOrgRoles` / `Config.CustomTeamRoles` (use `OrgRoles` / `TeamRoles`)
- New code should use the replacements noted on each symbol and `NewAuthService` / `aegis.New` for construction.

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

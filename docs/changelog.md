# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **OpenAPI Plugin**: New `openapi` plugin with automatic schema generation from `RouteMetadata`
- **Bearer Plugin**: Token-based authentication plugin with configurable validation
- **Testing Framework**: Comprehensive testing utilities in `core/testing.go` with thread-safe `MockDB`
- **Route Metadata System**: Automatic route documentation with request/response schema definitions
- **HTTP Helpers**: New validation and HTTP utility functions in `core/http_helpers.go` and `core/validation.go`
- **CLI Enhancements**: Expanded CLI with comprehensive plugin support and examples
- **Plugin Tests**: Test suites for admin, email, and SMS plugins demonstrating best practices
- **Schema Names**: Centralized OpenAPI schema name definitions across all plugins
- **Path Utilities**: Server path normalization and validation helpers
- **Database Helpers**: Enhanced database mapping utilities for plugin development
- **Documentation**: New guides for bearer plugin, OpenAPI plugin, password authentication, and database mapping
- **Example Projects**: Bearer and OpenAPI demonstration examples

### Changed
- **Plugin Architecture**: Complete refactor of plugin structure with consistent separation of concerns
  - All plugins now follow standardized structure: `handlers.go`, `models.go`, `db.go`, `schema_names.go`, `migrations.go`
  - Plugin-specific logic cleanly separated from core framework
- **Admin Plugin**: Major restructuring with middleware, utility functions, and comprehensive DB operations
- **Email Plugin**: Enhanced with proper database layer, test coverage, and schema definitions
- **SMS Plugin**: Restructured with dedicated DB layer and comprehensive test coverage
- **Organizations Plugin**: Significant expansion for multi-tenancy support
- **JWT Plugin**: Enhanced with schema definitions and improved handlers
- **OAuth Plugin**: Updated handlers with better error handling
- **Core Authentication**: Enhanced `core/auth.go` with improved user creation and session management
- **Session Management**: Refactored for better token handling and security
- **Password Hashing**: Improved Argon2id implementation with configurable parameters
- **Database Providers**: Enhanced SQL provider with better error handling and connection management
- **Router Implementations**: Updated Chi and default routers with improved route registration
- **Core Middleware**: Enhanced with better error handling and logging
- **Migration System**: Improved schema with better plugin support
- **Testing Helpers**: Enhanced test utilities with better mock implementations
- **Documentation**: Comprehensive updates to all guides including:
  - Getting Started guide significantly expanded
  - Core Concepts documentation enhanced with detailed explanations
  - Database Setup guide restructured with provider-specific details
  - ID Generation guide updated with ULID as default strategy
  - CLI documentation expanded with all plugin examples
  - Plugin documentation improved across all plugins
  - Testing guide enhanced with plugin testing examples
- **README**: Updated with better project overview and quick start instructions

### Deprecated
- None

### Removed
- **Password Plugin**: Functionality moved into core authentication (`core/password.go`)
- **Rate Limit Plugin**: Removed `plugins/ratelimit/ratelimit.go` (to be reimplemented)
- **Legacy Files**: Removed `core/token_provider.go` and `core/types.go` (consolidated into other modules)
- **Old Documentation**: Removed redundant docs (`architecture.md`, `configuration.md`, `examples-custom-id.md`, `plugins/password.md`)
- **SQLC Configuration**: Removed unused `migrations/sqlc/` directory
- **Goose Migrations**: Removed `migrations/goose/00001_aegis_core.sql` (consolidated into `migrations/schema.sql`)

### Fixed
- **Concurrent Map Access**: Fixed data race in `MockDB` with proper mutex synchronization
- **Plugin Initialization**: Improved plugin registration with "register-first + rollback-on-failure" pattern
- **Test Failures**: Resolved signup test failures in email and SMS plugins
- **Build Issues**: Fixed all `golangci-lint` warnings including:
  - `errcheck` warnings for unchecked errors
  - `gosec` security warnings
  - `ineffassign` ineffective assignments
  - `revive` code style issues
  - `staticcheck` code correctness issues
- **Type Safety**: Resolved integer overflow conversion warnings in ID generation
- **Import Paths**: Corrected module import paths throughout the codebase

### Security
- **Argon2id Password Hashing**: Industry-standard password hashing with configurable parameters
- **Secure Session Tokens**: Cryptographically secure token generation
- **JWT Signing**: Proper JWT validation and JWKS support
- **Input Validation**: Enhanced validation helpers to prevent common vulnerabilities
- **Thread Safety**: Proper synchronization in concurrent operations

## [1.0.0] - TBD

Initial release (planned)

### Added
- Complete authentication framework
- Plugin architecture
- CLI migration tool
- Comprehensive documentation

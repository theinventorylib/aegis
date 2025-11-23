# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- CLI tool for exporting migrations (`cmd/aegis`)
- Support for SQL, Goose, and golang-migrate export formats
- Email verification plugin
- SMS verification plugin
- OAuth plugin with Goth integration
- Password plugin with Argon2id hashing
- Organizations plugin for multi-tenancy
- Admin plugin for administrative APIs
- Core schema with 5 essential tables
- Session management with refresh tokens
- JWT support with JWKS
- PostgreSQL database provider
- MySQL database provider
- Plugin system with auto-discovery
- Migration export for Goose format
- Migration export for golang-migrate format
- Migration export for raw SQL

### Changed
- Refactored user model to minimal core fields
- Moved email, phone, and role to plugins
- Updated database provider interface for plugin compatibility

### Deprecated
- None

### Removed
- None

### Fixed
- None

### Security
- Argon2id password hashing
- Secure session token generation
- JWT signing and validation

## [1.0.0] - TBD

Initial release (planned)

### Added
- Complete authentication framework
- Plugin architecture
- CLI migration tool
- Comprehensive documentation

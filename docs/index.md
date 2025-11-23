# Aegis

**Aegis** is a lightweight, production-ready authentication framework for Go with a **modular plugin architecture**. It provides both a library for integration into your applications and a CLI tool for migration management.

[![CI](https://github.com/theinventorylib/aegis/actions/workflows/ci.yml/badge.svg)](https://github.com/theinventorylib/aegis/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/theinventorylib/aegis.svg)](https://pkg.go.dev/github.com/theinventorylib/aegis)

## Key Features

### 🛡️ Core Authentication
- **Minimal Core Schema**: Only 5 essential tables.
- **Session Management**: Secure token-based sessions with refresh tokens.
- **JWT Support**: Built-in JWKS for token signing and verification.
- **Database Agnostic**: Works with PostgreSQL, MySQL, SQLite, and more.

### 🔌 Plugin System
- **Email & SMS**: Verification via OTP or magic links.
- **OAuth**: Social login support (Google, GitHub, etc.).
- **Password**: Secure Argon2id hashing.
- **Organizations**: Multi-tenant support.
- **Custom Plugins**: Easily extensible architecture.

### 🛠️ Developer Experience
- **No Auto-Migration**: You maintain control over your database.
- **CLI Tools**: Export migrations to Goose, golang-migrate, or SQL.
- **Type-Safe**: Fully typed Go API.

## Documentation Sections

- [**Getting Started**](./getting-started.md): Installation and quick start guide.
- [**Core Concepts**](./core-concepts.md): Architecture, database provider, and schema.
- [**Architecture & Flow**](./architecture.md): Deep dive into how Aegis works.
- [**Configuration**](./configuration.md): Configuring Aegis and its plugins.
- [**Plugins**](./plugins.md): Detailed guides for available plugins.
- [**CLI Reference**](./cli.md): Using the Aegis CLI tool.
- [**Database Setup**](./database-setup.md): detailed database setup guide.

## Project Info

- [**Roadmap**](./roadmap.md): Future plans and upcoming features.
- [**Changelog**](./changelog.md): Version history and changes.
- [**Release Process**](./release-process.md): How we handle releases.

## Contributing

We welcome contributions! Please see our [Contributing Guide](./contributing.md) for details.

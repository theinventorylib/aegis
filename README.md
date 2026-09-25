# Aegis

<p align="center">
  <img src="assets/logo.png" width="300" alt="Aegis Logo">
</p>

**Aegis** is a lightweight authentication framework for Go with a **modular plugin architecture** inspired by [better-auth](https://www.better-auth.com/).

[![CI](https://github.com/theinventorylib/aegis/actions/workflows/ci.yml/badge.svg)](https://github.com/theinventorylib/aegis/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/theinventorylib/aegis/v2.svg)](https://pkg.go.dev/github.com/theinventorylib/aegis/v2)

---

## ✨ Features

### Core Authentication
- **Minimal Core Schema**: Only 4 essential tables
- **Database Agnostic**: Works with PostgreSQL, MySQL, SQLite
- **Session Management**: Secure token-based sessions with refresh tokens and Redis caching
- **CSRF Protection**: Built-in CSRF protection for web applications
- **Password Authentication**: Argon2id hashing built into core (not a plugin)
- **Developer Friendly**: No auto-migration magic, fully typed API

### 8 Official Plugins
- **Email** - Email verification via OTP and email+password auth
- **SMS** - Phone number verification via OTP
- **OAuth** - Social login (Google, GitHub, and more)
- **JWT** - Token generation, validation, and rotation
- **Admin** - Administrative endpoints for user management
- **Organizations** - Multi-tenant organization and team support
- **TOTP** - Time-based one-time password (RFC 6238) two-factor authentication
- **OpenAPI** - Interactive API documentation with Scalar UI

### Built-in Features
- **Bearer Auth** - Token authentication via `Authorization` header (config option, auto-enabled in API mode)

### CLI Tool
- **Migration Export**: Export database migrations in multiple formats
- **Format Support**: SQL, Goose, golang-migrate
- **Plugin Selection**: Export core + specific plugins or all at once

## 🚀 Quick Install

```bash
go get github.com/theinventorylib/aegis/v2
```

For the CLI tool:

```bash
go install github.com/theinventorylib/aegis/v2/cmd/aegis@latest
```

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](./CONTRIBUTING.md) for the branch model (v1 maintenance vs v2 development), [.github/COMMIT_GUIDE.md](./.github/COMMIT_GUIDE.md) for commit conventions, and [.github/RELEASE.md](./.github/RELEASE.md) for the release process.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details.

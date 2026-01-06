# Aegis

<p align="center">
  <img src="assets/logo.png" width="300" alt="Aegis Logo">
</p>

**Aegis** is a lightweight authentication framework for Go with a **modular plugin architecture** inspired by [better-auth](https://www.better-auth.com/).

[![CI](https://github.com/theinventorylib/aegis/actions/workflows/ci.yml/badge.svg)](https://github.com/theinventorylib/aegis/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/theinventorylib/aegis.svg)](https://pkg.go.dev/github.com/theinventorylib/aegis)

---

## ✨ Features

### Core Authentication
- **Minimal Core Schema**: Only 4 essential tables
- **Database Agnostic**: Works with PostgreSQL, MySQL, SQLite
- **Session Management**: Secure JWT-based sessions with refresh tokens
- **CSRF Protection**: Built-in CSRF protection for web applications
- **Password Authentication**: Argon2id hashing built into core (not a plugin)
- **Developer Friendly**: No auto-migration magic, fully typed API

### 8 Official Plugins
- **Email** - Email verification via OTP or magic links
- **SMS** - Phone number verification via OTP
- **OAuth** - Social login (Google, GitHub, and more)
- **JWT** - Token generation, validation, and rotation
- **Bearer** - Bearer token authentication support
- **Admin** - Administrative endpoints for user management
- **Organizations** - Multi-tenant organization and team support
- **OpenAPI** - Interactive API documentation with Scalar UI

### CLI Tool
- **Migration Export**: Export database migrations in multiple formats
- **Format Support**: SQL, Goose, golang-migrate
- **Plugin Selection**: Export core + specific plugins or all at once

## 🚀 Quick Install

```bash
go get github.com/theinventorylib/aegis
```

For the CLI tool:

```bash
go install github.com/theinventorylib/aegis/cmd/aegis@latest
```

## 🤝 Contributing

We welcome contributions! See [.github/COMMIT_GUIDE.md](./.github/COMMIT_GUIDE.md) for commit conventions and [.github/RELEASE.md](./.github/RELEASE.md) for release process.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details.

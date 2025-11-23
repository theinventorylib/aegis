# Aegis

[![CI](https://github.com/theinventorylib/aegis/actions/workflows/ci.yml/badge.svg)](https://github.com/theinventorylib/aegis/actions/workflows/ci.yml)
[![CodeQL](https://github.com/theinventorylib/aegis/actions/workflows/codeql.yml/badge.svg)](https://github.com/theinventorylib/aegis/actions/workflows/codeql.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/theinventorylib/aegis)](https://goreportcard.com/report/github.com/theinventorylib/aegis)
[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Reference](https://pkg.go.dev/badge/github.com/theinventorylib/aegis.svg)](https://pkg.go.dev/github.com/theinventorylib/aegis)

A lightweight, production-ready authentication framework for Go with a **modular plugin architecture**. Aegis provides both a library for integration into your applications and a CLI tool for migration management.

## 📦 Two Ways to Use Aegis

### As a Library (Primary Use Case)
Import Aegis into your Go application for full authentication capabilities:

```bash
go get github.com/theinventorylib/aegis
```

### As a CLI Tool (Migration Export)
Install the Aegis CLI for exporting database migrations:

```bash
go install github.com/theinventorylib/aegis/cmd/aegis@latest
```

---

## ✨ Features

### Core Authentication
- ✅ **Minimal Core Schema** - Only 5 essential tables (`user`, `accounts`, `verification`, `session`, `jwks`)
- ✅ **Session Management** - Token-based with refresh tokens
- ✅ **JWT Support** - JWKS for token signing and verification
- ✅ **Password Authentication** - Argon2 hashing via password plugin
- ✅ **Extensible** - Plugin architecture for additional auth methods

### Plugin System
- 🔌 **Email Verification** - Email OTP and magic links  
- 🔌 **SMS Verification** - Phone number OTP verification
- 🔌 **OAuth Support** - Social login (Google, GitHub, Apple, etc.)
- 🔌 **Password Plugin** - Secure password management
- 🔌 **Organizations** - Multi-tenant organization support
- 🔌 **Admin** - Administrative dashboard APIs
- 🔌 **Custom Plugins** - Create your own authentication methods

### Security
- 🔐 Argon2id password hashing
- 🔐 Secure session token generation
- 🔐 JWT signing and validation
- 🔐 Rate limiting ready
- 🔐 CSRF protection ready

### Developer Experience
- 📦 **No Auto-Migration** - You control your database
- 📦 **CLI Migration Export** - Export to Goose, golang-migrate, or raw SQL
- 📦 **Framework Agnostic** - Works with any Go web framework
- 📦 **Plugin Auto-Discovery** - Plugins register themselves
- 📦 **Type-Safe** - Fully typed Go code
- 📦 **Database Agnostic** - PostgreSQL and MySQL support

---

## 🚀 Quick Start

### 1. Install the Library

```bash
go get github.com/theinventorylib/aegis
```

### 2. Install the CLI (for migrations)

```bash
go install github.com/theinventorylib/aegis/cmd/aegis@latest
```

### 3. Export and Run Migrations

```bash
# Export migrations in Goose format
aegis export --format goose --output ./migrations/aegis --plugins email,sms,oauth

# Run migrations with Goose
goose -dir ./migrations/aegis postgres $DATABASE_URL up
```

**Supported migration formats:**
- `sql` - Plain SQL files
- `goose` - [Goose](https://github.com/pressly/goose) format
- `golang-migrate` - [golang-migrate](https://github.com/golang-migrate/migrate) format

See [migrations/README.md](./migrations/README.md) for detailed migration guide.

### 4. Initialize Aegis in Your Application

```go
package main

import (
    "github.com/theinventorylib/aegis"
    "github.com/theinventorylib/aegis/db"
    "github.com/theinventorylib/aegis/plugins/email"
    "github.com/theinventorylib/aegis/plugins/password"
)

func main() {
    // Initialize database
    database, _ := db.NewPostgresProvider("postgres://user:pass@localhost/db")
    
    // Create Aegis instance
    auth, _ := aegis.New(
        aegis.WithDatabase(database),
        aegis.WithJWTSecret("your-secret-key"),
    )
    
    // Add plugins
    passwordPlugin := password.New(&password.Config{DB: database})
    emailPlugin := email.New(&email.Config{
        DB: database,
        Provider: yourEmailProvider, // Implement email.Provider
        PasswordPlugin: passwordPlugin,
    })
    
    auth.Use(passwordPlugin)
    auth.Use(emailPlugin)
    
    // Initialize (registers routes)
    auth.Init()
    
    // Get router and start server
    router := auth.GetRouter()
    http.ListenAndServe(":8080", router)
}
```

---

## 📚 Documentation

### Core Documentation
- **[Getting Started Guide](./docs/getting-started.md)** - Complete setup walkthrough
- **[Migration Guide](./migrations/README.md)** - Database migration strategies
- **[API Reference](https://pkg.go.dev/github.com/theinventorylib/aegis)** - Go package documentation

### Plugin Documentation
- **[Email Plugin](./plugins/email/README.md)** - Email verification and authentication
- **[SMS Plugin](./plugins/sms/README.md)** - Phone number verification
- **[OAuth Plugin](./plugins/oauth/README.md)** - Social login integration
- **[Password Plugin](./plugins/password/README.md)** - Password authentication
- **[Creating Custom Plugins](./docs/custom-plugins.md)** - Build your own plugins

### CLI Documentation
- **[CLI Reference](./cmd/aegis/README.md)** - Migration export tool

---

## 🗄️ Core Schema

Aegis uses a **minimal core schema** with only 5 tables:

1. **`auth.user`** - User identities
2. **`auth.accounts`** - Authentication accounts (passwords, OAuth connections)  
3. **`auth.verification`** - Generic verification tokens
4. **`auth.session`** - User sessions with refresh tokens
5. **`auth.jwks`** - JSON Web Key Sets for JWT signing

**Plugins manage their own schemas** (e.g., `plugins_sms`, `plugins_email`, `plugins_oauth`).

---

## 🔌 Plugin System

### Available Plugins

| Plugin | Purpose | Tables |
|--------|---------|--------|
| **email** | Email verification | Adds `email`, `email_verified` to `auth.user` |
| **sms** | Phone verification | Adds `phone_number`, `phone_verified` to `auth.user` |
| **oauth** | Social login | `plugins_oauth.connections` |
| **password** | Password auth | Uses `auth.accounts` |
| **organizations** | Multi-tenancy | `auth.organizations`, `auth.user_organizations`, etc. |
| **admin** | Admin APIs | Queries existing tables |

### Using Plugins

```go
import (
    "github.com/theinventorylib/aegis/plugins/email"
    "github.com/theinventorylib/aegis/plugins/sms"
    "github.com/theinventorylib/aegis/plugins/oauth"
)

// Create and register plugins
emailPlugin := email.New(&email.Config{DB: db, Provider: emailProvider})
smsPlugin := sms.New(&sms.Config{DB: db, Provider: smsProvider})
oauthPlugin := oauth.New(&oauth.Config{DB: db})

auth.Use(emailPlugin)
auth.Use(smsPlugin)
auth.Use(oauthPlugin)
```

### Creating Custom Plugins

Plugins implement the `plugins.Plugin` interface:

```go
type Plugin interface {
    Name() string
    Version() string
    Description() string
    
    Init(ctx context.Context, a Aegis) error
    GetMigrations() []Migration
    MountRoutes(router server.Router, prefix string)
    
    Dependencies() []Dependency
    RequiresTables() []string
    ProvidesAuthMethods() []string
}
```

See [examples/custom-plugin](./examples/custom-plugin) for a complete example.

---

## 🛠️ CLI Usage

### Install

```bash
go install github.com/theinventorylib/aegis/cmd/aegis@latest
```

### Export Migrations

```bash
# Export all migrations (core + all plugins)
aegis export --format goose --output ./migrations

# Export specific plugins only
aegis export --format sql --plugins email,sms --output ./migrations

# Export core schema only
aegis export --core-only --output ./migrations

# Export for golang-migrate
aegis export --format golang-migrate --output ./migrations
```

### CLI Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--format` | Export format: `sql`, `goose`, `golang-migrate` | `sql` |
| `--output` | Output directory | `./aegis-migrations` |
| `--plugins` | Comma-separated plugins or `all` | `all` |
| `--core-only` | Export only core schema | `false` |

---

## 🏗️ Project Structure

```
aegis/
├── cmd/
│   └── aegis/           # CLI tool for migration export
├── core/                # Core authentication logic
├── db/                  # Database provider interfaces
│   ├── postgres.go      # PostgreSQL implementation
│   └── mysql.go         # MySQL implementation
├── migrations/          # Core schema and migration tools
│   ├── schema.sql       # Core schema definition
│   ├── exporter.go      # Migration export logic
│   └── README.md        # Migration documentation
├── models/              # Core data models
├── plugins/             # Plugin system
│   ├── email/           # Email verification plugin
│   ├── sms/             # SMS verification plugin
│   ├── oauth/           # OAuth plugin
│   ├── password/        # Password plugin
│   ├── organizations/   # Organizations plugin
│   └── admin/           # Admin plugin
├── server/              # HTTP server abstraction
├── examples/            # Example applications
│   ├── basic/           # Basic setup example
│   ├── plugins_demo/    # All plugins demo
│   └── custom-plugin/   # Custom plugin example
└── docs/                # Documentation
```

---

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](./CONTRIBUTING.md) for details.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/theinventorylib/aegis.git
cd aegis

# Install dependencies
go mod download

# Run tests
go test ./...

# Build CLI
cd cmd/aegis && go build
```

---

## 📋 Requirements

- **Go**: 1.25 or higher
- **Database**: PostgreSQL 12+ or MySQL 8+
- **Optional**: Migration tool (Goose, golang-migrate, etc.)

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details.

---

## 🙏 Acknowledgments

- [Goth](https://github.com/markbates/goth) - OAuth provider support
- [JWX](https://github.com/lestrrat-go/jwx) - JWT handling
- [Chi](https://github.com/go-chi/chi) - Default router (optional)
- [pgx](https://github.com/jackc/pgx) - PostgreSQL driver

---

## 📞 Support

- **Documentation**: [pkg.go.dev/github.com/theinventorylib/aegis](https://pkg.go.dev/github.com/theinventorylib/aegis)
- **Issues**: [GitHub Issues](https://github.com/theinventorylib/aegis/issues)
- **Discussions**: [GitHub Discussions](https://github.com/theinventorylib/aegis/discussions)

---

## 🗺️ Roadmap

- [ ] Atlas migration support
- [ ] MongoDB support
- [ ] WebAuthn/Passkeys plugin
- [ ] SAML plugin
- [ ] Rate limiting plugin
- [ ] Audit log plugin
- [ ] Account linking improvements
- [ ] CLI scaffolding commands

---

**Made with ❤️ for the Go community**

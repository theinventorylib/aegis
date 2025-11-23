# Contributing to Aegis

Thank you for your interest in contributing to Aegis! We welcome contributions from the community.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How Can I Contribute?](#how-can-i-contribute)
- [Development Setup](#development-setup)
- [Pull Request Process](#pull-request-process)
- [Coding Guidelines](#coding-guidelines)
- [Plugin Development](#plugin-development)
- [Testing](#testing)

## Code of Conduct

This project adheres to a code of conduct. By participating, you are expected to uphold this code. Please be respectful and constructive in all interactions.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check existing issues. When creating a bug report, include:

- **Clear title and description**
- **Steps to reproduce**
- **Expected behavior**
- **Actual behavior**
- **Go version** (`go version`)
- **Database version** (PostgreSQL/MySQL)
- **Code samples** if applicable

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, include:

- **Use case description**
- **Proposed solution**
- **Alternative solutions** considered
- **Impact on existing features**

### Pull Requests

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests
5. Run tests (`go test ./...`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

## Development Setup

### Prerequisites

- Go 1.25 or higher
- PostgreSQL 12+ or MySQL 8+ (for testing)
- Git

### Setup Steps

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/aegis.git
cd aegis

# Add upstream remote
git remote add upstream https://github.com/theinventorylib/aegis.git

# Install dependencies
go mod download

# Run tests
go test ./...

# Build CLI
cd cmd/aegis
go build -o aegis
./aegis --help
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./core
go test ./plugins/email

# Run with verbose output
go test -v ./...
```

### Database Setup for Testing

```bash
# PostgreSQL
createdb aegis_test
export DATABASE_URL="postgres://user:pass@localhost/aegis_test?sslmode=disable"

# MySQL
mysql -e "CREATE DATABASE aegis_test"
export DATABASE_URL="mysql://user:pass@localhost/aegis_test"
```

## Pull Request Process

1. **Update documentation** for any new features
2. **Add tests** for new functionality
3. **Ensure all tests pass** (`go test ./...`)
4. **Update CHANGELOG.md** if applicable
5. **Follow Go conventions** (run `go fmt`, `go vet`)
6. **Keep commits atomic** - one logical change per commit
7. **Write clear commit messages**

### Commit Message Format

```
type: subject

body (optional)

footer (optional)
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `refactor`: Code refactoring
- `test`: Adding tests
- `chore`: Maintenance tasks

**Examples:**
```
feat: add WebAuthn plugin for passwordless authentication

fix: correct session expiry calculation in refresh token flow

docs: update OAuth plugin README with GitHub provider example
```

## Coding Guidelines

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Run `go vet` and `golangci-lint` before committing
- Keep functions small and focused
- Write self-documenting code with clear names

### Package Organization

```
aegis/
├── core/           # Core authentication logic
├── db/             # Database interfaces and implementations
├── models/         # Shared data models
├── plugins/        # Plugin system and built-in plugins
│   └── myplugin/   # Each plugin is self-contained
│       ├── plugin.go      # Plugin implementation
│       ├── handlers.go    # HTTP handlers
│       ├── db.go          # Database operations
│       ├── migrations.go  # Migration definitions
│       └── README.md      # Plugin documentation
├── server/         # Server abstraction
└── cmd/            # CLI tools
```

### Error Handling

```go
// ✅ Good: Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to create user: %w", err)
}

// ❌ Bad: Lose error context
if err != nil {
    return err
}
```

### Testing

```go
// ✅ Good: Table-driven tests
func TestCreateUser(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    *User
        wantErr bool
    }{
        {"valid email", "user@example.com", &User{Email: "user@example.com"}, false},
        {"invalid email", "invalid", nil, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := CreateUser(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
            }
            // ... assertions
        })
    }
}
```

### Documentation

- **Package-level** docs for all packages
- **Function-level** docs for exported functions
- **Example code** for complex features
- **README files** for plugins

```go
// Package email provides email verification and authentication functionality.
//
// The email plugin adds email-based authentication to Aegis, including:
//   - Email verification via OTP or magic links
//   - Password reset flows
//   - Email + password login
//
// Example:
//
//    emailPlugin := email.New(&email.Config{
//        DB: database,
//        Provider: smtpProvider,
//    })
//    auth.Use(emailPlugin)
package email
```

## Plugin Development

### Creating a New Plugin

1. Create plugin directory: `plugins/myplugin/`
2. Implement `plugins.Plugin` interface
3. Add migrations via `GetMigrations()`
4. Add HTTP handlers
5. Write tests
6. Document in README

### Plugin Template

```go
package myplugin

import (
    "context"
    "github.com/theinventorylib/aegis/plugins"
    "github.com/theinventorylib/aegis/server"
)

type Plugin struct {
    db db.DBProvider
}

func New(db db.DBProvider) *Plugin {
    return &Plugin{db: db}
}

func (p *Plugin) Name() string { return "myplugin" }
func (p *Plugin) Version() string { return "1.0.0" }
func (p *Plugin) Description() string { return "My custom plugin" }

func (p *Plugin) Init(ctx context.Context, a plugins.Aegis) error {
    // Initialize plugin
    return nil
}

func (p *Plugin) GetMigrations() []plugins.Migration {
    return []plugins.Migration{{
        Version: "001",
        Description: "Create myplugin schema",
        Up: `CREATE SCHEMA IF NOT EXISTS plugins_myplugin;`,
        Down: `DROP SCHEMA IF EXISTS plugins_myplugin CASCADE;`,
    }}
}

func (p *Plugin) MountRoutes(router server.Router, prefix string) {
    router.POST(prefix+"/myplugin/action", p.handleAction)
}

func (p *Plugin) RequiresTables() []string { return []string{"auth.user"} }
func (p *Plugin) ProvidesAuthMethods() []string { return []string{"myplugin"} }
func (p *Plugin) Dependencies() []plugins.Dependency { return nil }
```

### Plugin Guidelines

- **Self-contained**: Plugins should not depend on other plugins
- **Namespaced**: Use `plugins_[name]` schema for plugin tables
- **Migrations**: Always provide both Up and Down migrations
- **Documentation**: Include README with usage examples
- **Testing**: Write comprehensive tests

## Testing

### Unit Tests

```bash
# Run unit tests
go test ./core
go test ./plugins/email
```

### Integration Tests

```bash
# Requires running database
export DATABASE_URL="postgres://localhost/aegis_test"
go test -tags=integration ./...
```

### CLI Tests

```bash
# Build and test CLI
cd cmd/aegis
go build
./aegis export --help
./aegis export --core-only --output /tmp/test
```

## Questions?

- **Documentation**: Check [pkg.go.dev](https://pkg.go.dev/github.com/theinventorylib/aegis)
- **Discussions**: Use [GitHub Discussions](https://github.com/theinventorylib/aegis/discussions)
- **Issues**: Open an [issue](https://github.com/theinventorylib/aegis/issues)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

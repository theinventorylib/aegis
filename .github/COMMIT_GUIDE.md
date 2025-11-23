# Git Commit Guide for Aegis

## Commit Message Format

We follow [Conventional Commits](https://www.conventionalcommits.org/) for clear, structured commit messages.

### Structure

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Type

**Required.** Must be one of:

- **feat**: New feature
- **fix**: Bug fix
- **docs**: Documentation only changes
- **style**: Code style (formatting, missing semi-colons, etc.)
- **refactor**: Code change that neither fixes a bug nor adds a feature
- **perf**: Performance improvement
- **test**: Adding or updating tests
- **chore**: Maintenance tasks (dependencies, build config, etc.)
- **ci**: CI/CD changes

### Scope

**Optional.** The area of the codebase:

- `core` - Core authentication
- `plugins` - Plugin system
- `email` - Email plugin
- `sms` - SMS plugin
- `oauth` - OAuth plugin
- `password` - Password plugin
- `cli` - CLI tool
- `db` - Database layer
- `migrations` - Migration system
- `ci` - CI/CD
- `docs` - Documentation

### Subject

**Required.** Brief description:

- Use imperative mood ("add" not "added")
- Don't capitalize first letter
- No period at the end
- Max 50 characters

### Body

**Optional.** Detailed description:

- Use imperative mood
- Wrap at 72 characters
- Explain what and why, not how

### Footer

**Optional.** Reference issues, breaking changes:

- **Breaking Changes**: `BREAKING CHANGE: description`
- **Issues**: `Fixes #123`, `Closes #456`

---

## Examples

### Feature Addition

```bash
git commit -m "feat(email): add magic link authentication

Implement passwordless authentication via email magic links.
Users receive a one-time link that logs them in without a password.

- Add magic link generation with expiry
- Add email template for magic links
- Add verification endpoint

Closes #42"
```

### Bug Fix

```bash
git commit -m "fix(oauth): correct callback URL validation

OAuth callback was failing for localhost URLs with ports.
Updated regex to allow port numbers in callback validation.

Fixes #89"
```

### Documentation

```bash
git commit -m "docs(cli): update migration export examples

Add examples for all three export formats (sql, goose, golang-migrate)
with clear usage instructions for each migration tool."
```

### Refactoring

```bash
git commit -m "refactor(core): extract session validation logic

Move session validation from handlers to dedicated service.
Improves testability and code organization."
```

### Breaking Change

```bash
git commit -m "feat(core): redesign user schema

BREAKING CHANGE: User model no longer includes email, phone, or role fields.
These fields are now managed by respective plugins.

Migration guide in docs/migration-v2.md"
```

### Chore

```bash
git commit -m "chore(deps): update dependencies

Update go.mod dependencies to latest versions.
Run go mod tidy to clean up unused dependencies."
```

### CI/CD

```bash
git commit -m "ci: add CodeQL security scanning

Add weekly CodeQL workflow for vulnerability detection.
Scans run on PRs and every Monday."
```

---

## Initial Commit Strategy

For your first commit, organize files logically:

### Step 1: Initial Commit (Core Structure)

```bash
git add go.mod go.sum
git add aegis.go
git add LICENSE README.md
git commit -m "chore: initial project setup

Initialize Aegis authentication framework with Go module,
main package entry point, MIT license, and project README."
```

### Step 2: Core Implementation

```bash
git add core/ models/ db/ server/ config/
git commit -m "feat(core): implement core authentication system

Add core authentication functionality:
- User and session models
- Database provider interface (PostgreSQL, MySQL)
- Session service for token management
- Server abstraction for HTTP routing
- Configuration system

Includes 5-table minimal schema design."
```

### Step 3: Plugins

```bash
git add plugins/
git commit -m "feat(plugins): add plugin system and built-in plugins

Implement modular plugin architecture with 6 built-in plugins:
- email: Email verification and authentication
- sms: Phone number verification
- oauth: Social login with Goth
- password: Argon2id password hashing
- organizations: Multi-tenancy support
- admin: Admin API endpoints

Each plugin manages its own schema and provides migrations."
```

### Step 4: Migrations

```bash
git add migrations/
git commit -m "feat(migrations): add migration export system

Add CLI-based migration export with support for:
- SQL: Plain SQL files
- Goose: Goose migration format
- golang-migrate: golang-migrate format

Core schema embedded for easy distribution."
```

### Step 5: CLI Tool

```bash
git add cmd/
git commit -m "feat(cli): add migration export CLI tool

Add aegis CLI for exporting database migrations.
Supports plugin filtering, format selection, and core-only mode.

Usage: aegis export --format goose --plugins email,sms"
```

### Step 6: Examples

```bash
git add examples/
git commit -m "docs: add usage examples

Add example applications:
- basic: Simple authentication setup
- plugins_demo: All plugins demonstration
- oauth: OAuth integration example"
```

### Step 7: Documentation

```bash
git add docs/ CONTRIBUTING.md CHANGELOG.md
git commit -m "docs: add comprehensive documentation

Add project documentation:
- Contributing guidelines
- Changelog template
- ID generation guide
- Custom ID examples"
```

### Step 8: CI/CD

```bash
git add .github/ .golangci.yml .goreleaser.yml
git commit -m "ci: add GitHub Actions workflows and tooling

Add enterprise-grade CI/CD pipeline:
- CI: Matrix testing, linting, format checking
- Release: GoReleaser for multi-platform builds
- CodeQL: Security scanning
- Dependabot: Dependency updates
- golangci-lint: Comprehensive linting (20+ linters)"
```

### Step 9: Tooling Configuration

```bash
git add .gitignore
git commit -m "chore: add gitignore for Go project

Ignore build artifacts, IDE files, test data, and temporary files."
```

---

## Alternative: Single Initial Commit

If you prefer one large initial commit:

```bash
git add .
git commit -m "chore: initial commit - Aegis v1.0.0

Initialize Aegis authentication framework with complete implementation:

Core Features:
- 5-table minimal schema (user, accounts, verification, session, jwks)
- PostgreSQL and MySQL support
- Session management with refresh tokens
- JWT support with JWKS

Plugins:
- Email verification and authentication
- SMS verification
- OAuth social login
- Password authentication (Argon2id)
- Organizations (multi-tenancy)
- Admin APIs

Tools:
- CLI for migration export (SQL, Goose, golang-migrate)
- Plugin system with auto-discovery

CI/CD:
- GitHub Actions workflows
- GoReleaser for multi-platform releases
- CodeQL security scanning
- golangci-lint configuration

Documentation:
- Comprehensive README
- Contributing guidelines
- Usage examples
- Plugin development guide"
```

---

## Best Practices

### DO ✅

- Write clear, descriptive commit messages
- Keep commits focused (one logical change)
- Use conventional commit format
- Reference issues when applicable
- Test before committing
- Run `go fmt` before committing

### DON'T ❌

- Mix unrelated changes in one commit
- Write vague messages ("fix stuff", "updates")
- Commit broken code
- Commit commented-out code
- Commit sensitive data (.env files)

---

## Quick Reference

```bash
# Stage changes
git add <files>

# Commit with message
git commit -m "type(scope): subject"

# Amend last commit (if not pushed)
git commit --amend

# Add to last commit without changing message
git commit --amend --no-edit

# View commit history
git log --oneline

# View specific file history
git log --oneline -- <file>
```

---

## Recommended First Steps

For your current state, I recommend:

```bash
# 1. Add everything for initial commit
git add .

# 2. Initial commit
git commit -m "chore: initial commit - Aegis authentication framework v1.0.0

Complete authentication framework with core features, plugins, CLI tool,
and enterprise-grade CI/CD pipeline. Ready for v1.0.0-beta.1 release."

# 3. Create GitHub repository and push
git remote add origin https://github.com/theinventorylib/aegis.git
git branch -M main
git push -u origin main

# 4. Tag initial version
git tag -a v1.0.0-beta.1 -m "Beta release v1.0.0-beta.1"
git push origin v1.0.0-beta.1
```

This will trigger your release workflow and create the first release! 🚀

# Release Process

This document describes the release process for Aegis.

## Overview

Aegis uses automated releases via GitHub Actions and GoReleaser. When you push a version tag, the release workflow automatically:

1. Runs tests
2. Builds CLI binaries for multiple platforms
3. Creates GitHub release with changelog
4. Uploads release artifacts

## Creating a Release

### 1. Update Version Information

Update the version in relevant files:

```bash
# Update CHANGELOG.md
# Add new version section with changes

# Commit changes
git add CHANGELOG.md
git commit -m "chore: prepare release v1.0.0"
git push origin main
```

### 2. Create and Push Tag

```bash
# Create annotated tag
git tag -a v1.0.0 -m "Release v1.0.0"

# Push tag to trigger release workflow
git push origin v1.0.0
```

### 3. Monitor Release Workflow

Watch the [Actions tab](https://github.com/theinventorylib/aegis/actions) to monitor the release process:

1. Tests run
2. GoReleaser builds binaries
3. GitHub release is created

### 4. Verify Release

After the workflow completes:

1. Check the [Releases page](https://github.com/theinventorylib/aegis/releases)
2. Verify binaries are attached
3. Verify changelog is correct
4. Test installation:

```bash
# Test library installation
go get github.com/theinventorylib/aegis@v1.0.0

# Test CLI installation
go install github.com/theinventorylib/aegis/cmd/aegis@v1.0.0
aegis version
```

## Version Numbering

Aegis follows [Semantic Versioning](https://semver.org/):

- **MAJOR** version: incompatible API changes
- **MINOR** version: backwards-compatible functionality
- **PATCH** version: backwards-compatible bug fixes

Examples:
- `v1.0.0` - Initial release
- `v1.1.0` - New features (email plugin)
- `v1.1.1` - Bug fixes
- `v2.0.0` - Breaking changes (API redesign)

## Pre-releases

For beta or release candidate versions:

```bash
# Create pre-release tag
git tag -a v1.0.0-beta.1 -m "Beta release v1.0.0-beta.1"
git push origin v1.0.0-beta.1
```

GoReleaser will automatically mark these as pre-releases.

## Hotfix Releases

For urgent bug fixes:

1. Create hotfix branch from tag:
```bash
git checkout -b hotfix/v1.0.1 v1.0.0
```

2. Make fixes and commit

3. Create tag:
```bash
git tag -a v1.0.1 -m "Hotfix v1.0.1"
git push origin v1.0.1
```

4. Merge back to main:
```bash
git checkout main
git merge hotfix/v1.0.1
git push origin main
```

## Release Checklist

Before creating a release:

- [ ] All tests pass (`go test ./...`)
- [ ] Linting passes (`golangci-lint run`)
- [ ] CHANGELOG.md is updated
- [ ] Documentation is up to date
- [ ] Breaking changes are documented
- [ ] Examples are tested
- [ ] CLI builds successfully

After release:

- [ ] Verify GitHub release
- [ ] Test library installation
- [ ] Test CLI installation
- [ ] Update documentation if needed
- [ ] Announce release (optional)

## Automated Workflows

### CI Workflow

Runs on every push and pull request:
- Tests on multiple Go versions
- Linting with golangci-lint
- Format checking
- Build verification

### Release Workflow

Runs on version tag push:
- Runs tests
- Builds CLI for Linux, macOS, Windows (amd64 & arm64)
- Generates checksums
- Creates GitHub release
- Uploads artifacts

### CodeQL Workflow

Runs on schedule and pull requests:
- Security vulnerability scanning
- Code quality analysis

## GoReleaser Configuration

The `.goreleaser.yml` file configures:

- **Builds**: Multi-platform CLI binaries
- **Archives**: tar.gz for Linux/macOS, zip for Windows
- **Checksums**: SHA256 checksums file
- **Changelog**: Automated from commit messages
- **Release notes**: Custom template with installation instructions

## Troubleshooting

### Release Failed

Check the Actions logs for errors:

1. Go to [Actions tab](https://github.com/theinventorylib/aegis/actions)
2. Click on failed workflow
3. Review logs

Common issues:
- Tests failing
- Build errors
- GoReleaser configuration issues

### Tag Already Exists

If you need to recreate a tag:

```bash
# Delete local tag
git tag -d v1.0.0

# Delete remote tag
git push --delete origin v1.0.0

# Recreate tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

### Binary Not Building

Ensure `.goreleaser.yml` is correctly configured:

```bash
# Test GoReleaser locally
goreleaser release --snapshot --clean
```

## Manual Release (Emergency)

If automated release fails, you can release manually:

```bash
# Install GoReleaser
go install github.com/goreleaser/goreleaser@latest

# Create release
GITHUB_TOKEN="your_token" goreleaser release --clean
```

## See Also

- [CHANGELOG.md](../CHANGELOG.md) - Release history
- [CONTRIBUTING.md](../CONTRIBUTING.md) - Contribution guidelines
- [.goreleaser.yml](../.goreleaser.yml) - GoReleaser configuration

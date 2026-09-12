# Contributing to Aegis

Thanks for contributing. This repository maintains **two release lines in
parallel**, so the first question for any change is *which line does it belong
to?*

## Branch model

| Branch | Line | Module path | Purpose |
|--------|------|-------------|---------|
| `main` | v2 | `github.com/theinventorylib/aegis/v2` | Current major. Breaking changes and new features land here. |
| `v1` | v1 | `github.com/theinventorylib/aegis` | Maintenance line. Bug fixes and security patches only. |

Both are permanent lines. `v1` is not a snapshot — it is actively patched and
tagged independently.

## Where to send a change

- **Bug fix / security fix for the v1 API** → base your PR on `v1`. If the same
  bug exists on `main`, forward-port it there too (a separate PR or a cherry-pick
  once the v1 fix is merged).
- **New feature, or any change that alters the public API** → base it on `main`.
  If it is also needed on v1, that is a deliberate backport decision, not a
  default.
- **Never mix a v2-only change into a `v1` PR.** A stray v2 change on the v1 line
  silently reintroduces a breaking change into what users expect to be safe.

Working branches use the `gr1nch3/*` prefix and target the appropriate line.

## Releases

Releases are cut with the `Release` workflow (`workflow_dispatch`), selecting
`patch`, `minor`, or `major`. See [.github/RELEASE.md](./RELEASE.md) for the full
process.

The workflow validates the computed version against the branch:

- `main` must produce `v2+`
- `v1` must produce `v0`/`v1`

A `major` bump on `v1`, or a `minor`/`patch` bump on `main` before `v2.0.0`
exists, fails fast rather than minting a tag on the wrong line.

## Local checks

```bash
gofmt -l -s .          # formatting
go vet ./...           # vet
go test ./...          # tests
golangci-lint run ./... # lint
```

The `examples/*/` directories are **separate Go modules** (matched by the root
`replace` directive). Build them individually if you touch them:

```bash
for d in examples/*/; do (cd "$d" && go build ./...); done
```

## Changelog

[GitHub Releases](https://github.com/theinventorylib/aegis/releases) is the
single source of truth for the changelog. The docs site regenerates
`docs/content/changelog.md` from Releases at deploy time (see
`docs/scripts/sync-changelog.sh`), so do not hand-edit it for release notes.

## Commit conventions

See [.github/COMMIT_GUIDE.md](./COMMIT_GUIDE.md).

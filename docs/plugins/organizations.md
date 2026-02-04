# Organizations Plugin

The Organizations plugin adds multi-tenancy support to Aegis, allowing users to create and manage organizations and teams.

## Installation

```go
import "github.com/theinventorylib/aegis/plugins/organizations"
```

## Usage

```go
orgPlugin := organizations.New(nil, plugins.DialectPostgres)
aegis.UseWithPriority(ctx, orgPlugin, 10) // Initialize early
```

## Schema

This plugin creates the following tables:
- `organization`
- `members`
- `team`
- `team_member`

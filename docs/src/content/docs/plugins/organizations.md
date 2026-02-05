---
title: Organizations Plugin
---
## Organizations Plugin
Adds multi-tenancy support (Organizations and Teams).
```go
import "github.com/theinventorylib/aegis/plugins/organizations"
orgPlugin := organizations.New(nil, plugins.DialectPostgres)
aegis.UseWithPriority(ctx, orgPlugin, 10)
```

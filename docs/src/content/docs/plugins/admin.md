---
title: Admin Plugin
description: Manage users and sessions.
---

The Admin plugin provides endpoints for managing users, sessions, and system configuration.

## Installation

```go
import "github.com/theinventorylib/aegis/plugins/admin"
```

## Usage

```go
adminPlugin := admin.New(adminConfig)
aegis.Use(ctx, adminPlugin)
```

---
title: Getting Started
description: How to install and start using Aegis.
---

Aegis is a lightweight, modular authentication framework for Go. It provides the essential building blocks for secure authentication without forcing a specific database or router on you.

## Quick Install

To install the core framework:

```bash
go get github.com/theinventorylib/aegis
```

To install the CLI tool (for migration exports):

```bash
go install github.com/theinventorylib/aegis/cmd/aegis@latest
```

## Basic Usage

Aegis is designed to be easy to set up. Here is a minimal example of how to initialize Aegis with the built-in standard library `net/http` router (though it works with any router).

```go
package main

import (
    "context"
    "database/sql"
    "net/http"

    "github.com/theinventorylib/aegis"
    "github.com/theinventorylib/aegis/config"
    _ "github.com/lib/pq" // Postgres driver
)

func main() {
    // 1. Setup database connection
    db, _ := sql.Open("postgres", "postgres://user:pass@localhost/db?sslmode=disable")

    // 2. Configure Aegis
    cfg := config.Default().
        WithDB(db).
        WithSecret([]byte("your-32-byte-secret-key-here!!!!")).
        WithRedis("localhost", 6379, "", 0)

    // 3. Initialize Aegis
    auth, err := aegis.New(context.Background(), cfg)
    if err != nil {
        panic(err)
    }

    // 4. Mount routes
    // This will mount auth routes at /auth/*
    http.Handle("/auth/", auth.Handler())

    // 5. Start server
    http.ListenAndServe(":8080", nil)
}
```

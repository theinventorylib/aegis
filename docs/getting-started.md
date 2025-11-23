# Getting Started

This guide will help you get up and running with Aegis in your Go application.

## Prerequisites

- **Go**: Version 1.25 or higher.
- **Database**: A supported SQL database (PostgreSQL, MySQL, SQLite, etc.).

## Installation

### 1. Install the Library

Add Aegis to your project using `go get`:

```bash
go get github.com/theinventorylib/aegis
```

### 2. Install the CLI (Optional but Recommended)

The CLI tool is useful for exporting database migrations.

```bash
go install github.com/theinventorylib/aegis/cmd/aegis@latest
```

## Quick Start

### 1. Database Setup

First, you need to set up your database. Aegis does not automatically migrate your database at runtime. You must export the migrations and run them using your preferred migration tool.

```bash
# Export migrations to a directory
aegis export --format sql --output ./migrations/aegis

# Run the migrations (example using psql for PostgreSQL)
psql $DATABASE_URL -f ./migrations/aegis/001_core.up.sql
```

For more details, see the [Database Setup](./database-setup.md) guide.

### 2. Initialize Aegis

Here is a basic example of how to initialize Aegis with the standard `database/sql` package.

```go
package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq" // Import your database driver

	"github.com/theinventorylib/aegis"
	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/db"
)

func main() {
	// 1. Connect to your database
	dbConn, err := sql.Open("postgres", "postgres://user:pass@localhost:5432/myapp?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer dbConn.Close()

	// 2. Create a Router
	// Aegis requires a router adapter. We provide a default one for net/http.
	mux := http.NewServeMux()
	router := server.NewDefaultRouter(mux)

	// 3. Initialize Aegis
	auth, err := aegis.New(
		config.WithDB(dbConn, db.PostgreSQL),
		config.WithRouter(router),
		config.WithJWTSecret([]byte("your-very-secure-secret-key")),
	)
	if err != nil {
		log.Fatal("Failed to init Aegis:", err)
	}

	// 4. Mount Routes
	// This mounts all auth routes (e.g., /auth/login, /auth/user)
	auth.MountRoutes("") 

	// 5. Start Server
	// Since we used the mux in NewDefaultRouter, Aegis routes are registered there.
	fmt.Println("Server listening on :8080")
	http.ListenAndServe(":8080", mux)
}
```

### 3. Next Steps

- Explore [Core Concepts](./core-concepts.md) to understand how Aegis works.
- Check out the [Plugins](./plugins.md) to add email, SMS, or OAuth authentication.
- Read the [Configuration](./configuration.md) guide for advanced options.

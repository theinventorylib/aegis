# ID Generation Strategies Example

This example demonstrates how to configure different ID generation strategies in Aegis.

## Features

- ULID (Universally Unique Lexicographically Sortable Identifier) - default
- UUID (Universally Unique Identifier v4)
- Custom ID generators
- API endpoints to demonstrate ID generation
- Integration with user authentication (IDs used for users, sessions, etc.)

## Prerequisites

- Go 1.21 or higher
- PostgreSQL database
- Aegis CLI tool (for migrations)

## Setup

### 1. Install Dependencies

```bash
go mod init aegis-id-generation-example
go get github.com/theinventorylib/aegis
go get github.com/go-chi/chi/v5
go get github.com/lib/pq
```

### 2. Create Database

```bash
createdb aegis_id_gen
```

### 3. Export and Run Migrations

```bash
# Install Aegis CLI if not already installed
go install github.com/theinventorylib/aegis/cmd/aegis@latest

# Export migrations for PostgreSQL
aegis export --dialect postgres --output ./migrations

# Run migrations (example using psql)
psql aegis_id_gen < migrations/001_aegis_auth_schema.sql
```

### 4. Update Database Connection

Edit `main.go` and update the database connection string:

```go
db, err := sql.Open("postgres", "postgres://user:password@localhost/aegis_id_gen?sslmode=disable")
```

Replace `user:password` with your PostgreSQL credentials.

### 5. Run the Example

```bash
go run main.go
```

Visit http://localhost:8080 to see the example.

## ID Generation Strategies

### ULID (Default)

- **Length**: 26 characters
- **Sortable**: Yes (lexicographically)
- **Time-based**: Yes (first 10 characters are timestamp)
- **Format**: Base32-encoded, URL-safe
- **Example**: `01HXXXXXXXXXXXXXXXXXXXXX`

### UUID v4

- **Length**: 36 characters (including hyphens)
- **Sortable**: No
- **Time-based**: No (random)
- **Format**: Standard UUID format with hyphens
- **Example**: `550e8400-e29b-41d4-a716-446655440000`

### Custom Generator

You can provide your own ID generation function:

```go
config.WithIDGenerator(func() string {
    return "CUSTOM-" + someLogic()
})
```

## API Endpoints

- `GET /` - Home page with links
- `GET /ids` - JSON response with sample generated IDs
- `GET /strategy` - JSON response with strategy information
- `POST /auth/signup` - User registration
- `POST /auth/login` - User login
- `GET /auth/user` - Get current user (protected)
- `POST /auth/logout` - User logout
- `GET /dashboard` - Protected dashboard

## Changing ID Strategy

To use a different strategy, modify the Aegis configuration:

```go
// For UUID
aegis.New(ctx, config.WithIDStrategy(core.IDStrategyUUID), ...)

// For custom generator
aegis.New(ctx, config.WithIDGenerator(myGenerator), ...)
```

**Note**: Changing ID strategy after data exists may cause issues with foreign key references.

## Web Interface

The example includes a simple web interface to:

1. View current ID strategy
2. Generate sample IDs via API
3. Test authentication (user IDs will use the configured strategy)

## Database Schema

The example uses standard Aegis tables where IDs are used for:

- Users (`users` table)
- Sessions (`sessions` table)
- Any plugin-specific tables (OAuth states, etc.)

All ID columns use the configured generation strategy.</content>
<parameter name="filePath">/home/gr1nch3/Documents/TheInventory/projects/aegis/examples/05-id-generation/README.md
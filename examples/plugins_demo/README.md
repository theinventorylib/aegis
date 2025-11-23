# Aegis Plugins Demo

This example demonstrates how to properly configure and use Aegis authentication plugins including **Password**, **Email**, **SMS**, and **Admin** plugins.

## What This Example Shows

- ✅ Database-agnostic plugin initialization (works with PostgreSQL or MySQL)
- ✅ Proper plugin configuration with mock providers
- ✅ Plugin registration with Aegis core
- ✅ Route mounting for all plugin endpoints
- ✅ Protected routes using authentication middleware

## Prerequisites

- Go 1.21 or higher
- PostgreSQL or MySQL database running
- Database with Aegis core schema initialized

## Quick Start

### 1. Set Up Database

**PostgreSQL:**
```bash
createdb aegis_demo
psql aegis_demo < path/to/core_schema.sql
```

**MySQL:**
```bash
mysql -u root -p -e "CREATE DATABASE aegis_demo"
mysql -u root -p aegis_demo < path/to/core_schema.sql
```

### 2. Update Connection String

Edit `main.go` and update the database connection:

```go
// For PostgreSQL
connString := "postgres://user:password@localhost:5432/aegis_demo?sslmode=disable"
database, err := db.NewPostgresProvider(connString)

// For MySQL
connString := "user:password@tcp(127.0.0.1:3306)/aegis_demo?parseTime=true"
database, err := db.NewMySQLProvider(connString)
```

### 3. Run the Example

```bash
cd examples/plugins_demo
go run main.go
```

The server will start on `http://localhost:3001`

## Available Endpoints

### Email Plugin
- `POST /auth/email/login` - Login with email + password

### SMS Plugin
- `POST /auth/sms/send` - Send SMS OTP
- `POST /auth/sms/verify` - Verify SMS OTP
- `POST /auth/sms/login` - Login with phone + password

### Password Plugin
- `POST /auth/password/change` - Change user password

### Admin Plugin
- `GET /auth/admin/users` - List all users (admin only)
- `GET /auth/admin/users/:id` - Get user by ID
- `POST /auth/admin/users/:id/disable` - Disable user
- `POST /auth/admin/users/:id/enable` - Enable user
- `DELETE /auth/admin/users/:id` - Delete user
- `GET /auth/admin/organizations` - List organizations
- `GET /auth/admin/stats` - Get system statistics

### Protected Routes
- `GET /api/me` - Get current authenticated user

## Plugin Configuration

### Email Plugin

```go
emailPlugin := email.New(&email.Config{
    DB:             database,           // DBProvider instance
    Provider:       &MockEmailProvider{}, // Email sender implementation
    OTPExpiry:      15 * time.Minute,   // OTP code expiration
    PasswordPlugin: passwordPlugin,      // Optional: enable email+password auth
})
```

### SMS Plugin

```go
smsPlugin := sms.New(&sms.Config{
    DB:             database,          // DBProvider instance
    Provider:       &MockSMSProvider{}, // SMS sender implementation
    OTPExpiry:      5 * time.Minute,   // OTP code expiration
    OTPLength:      6,                  // OTP code length
    PasswordPlugin: passwordPlugin,     // Optional: enable phone+password auth
})
```

### Password Plugin

```go
passwordPlugin := password.New(&password.Config{
    DB:     database,                           // DBProvider instance
    UserDB: database,                           // User lookup provider
    Hasher: core.DefaultPasswordHasherConfig(), // Password hashing config
})
```

### Admin Plugin

```go
adminPlugin := admin.New(database) // DBProvider instance
```

## Mock Providers

This example uses mock email and SMS providers that print to console instead of actually sending messages. In production, replace these with real implementations:

**Email Providers:**
- SMTP (net/smtp)
- SendGrid
- AWS SES
- Resend
- Postmark

**SMS Providers:**
- Twie
- AWS SNS
- Vonage
- MessageBird

## Database Adapters

Aegis supports multiple databases through the `DBProvider` interface:

- ✅ **PostgreSQL** - `db.NewPostgresProvider(connString)`
- ✅ **MySQL** - `db.NewMySQLProvider(connString)`
- 🔜 SQLite, CockroachDB (coming soon)

## Testing

### Send SMS OTP
```bash
curl -X POST http://localhost:3001/auth/sms/send \
  -H "Content-Type: application/json" \
  -d '{"phoneNumber": "+1234567890", "purpose": "login_mfa"}'
```

### Verify SMS OTP
```bash
curl -X POST http://localhost:3001/auth/sms/verify \
  -H "Content-Type: application/json" \
  -d '{"phoneNumber": "+1234567890", "code": "123456", "purpose": "login_mfa"}'
```

### List Users (Admin)
```bash
curl http://localhost:3001/auth/admin/users \
  -H "Cookie: session_token=YOUR_SESSION_TOKEN"
```

## Plugin Migrations

Each plugin declares its own database migrations:

- **Email**: Adds `email` and `email_verified` columns to `auth.user`, creates `plugins_email_verifications` table
- **SMS**: Adds `phone_number` and `phone_verified` columns to `auth.user`, creates `plugins_sms_verifications` table
- **Password**: Uses core `auth.accounts` table
- **Admin**: No custom tables (uses core schema)
- **Organizations**: Creates organization and team management tables

Migrations should be run via a migration tool (coming soon).

## Architecture

This example demonstrates the plugin-based architecture of Aegis:

1. **Core** provides base authentication (sessions, CSRF)
2. **Plugins** extend functionality (email, SMS, password, admin)
3. **DBProvider** abstracts database operations (PostgreSQL, MySQL)
4. **Shared Utilities** (`core.GenerateOTPCode`, `core.GenerateID`)

## Learn More

- [Main Documentation](../../README.md)
- [Plugin Development Guide](../../docs/plugins.md)
- [Database Abstraction](../../docs/database.md)

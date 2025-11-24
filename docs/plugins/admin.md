# Admin Plugin

The Admin plugin provides administrative endpoints for managing users, organizations, and viewing system statistics.

**SECURITY WARNING**: All endpoints in this plugin are protected and require authentication. In a production environment, you should also ensure that the authenticated user has appropriate administrative privileges (e.g., via a `RequireAdmin` middleware).

## Installation

```bash
go get github.com/theinventorylib/aegis/plugins/admin
```

## Configuration

```go
import "github.com/theinventorylib/aegis/plugins/admin"

adminPlugin := admin.New(&admin.Config{
    DB:             dbProvider,
    SessionService: sessionService,
})
```

## Endpoints

All endpoints require a valid session cookie or Bearer token.

### User Management
- `GET /admin/users`: List all users.
- `GET /admin/users/:id`: Get details for a specific user.
- `POST /admin/users/:id/disable`: Disable a user account.
- `POST /admin/users/:id/enable`: Enable a user account.
- `DELETE /admin/users/:id`: Permanently delete a user.

### Organization Management
- `POST /admin/organizations`: Create a new organization.
- `GET /admin/organizations`: List all organizations.
- `GET /admin/organizations/:id`: Get details for a specific organization.
- `POST /admin/organizations/:id/ban`: Ban an organization.
- `DELETE /admin/organizations/:id`: Delete an organization.

### System
- `GET /admin/stats`: View system statistics.

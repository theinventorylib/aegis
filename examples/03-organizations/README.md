# Organizations (Multi-Tenant) Example

This example demonstrates how to build a multi-tenant SaaS application with Aegis organizations plugin.

## Features

- Multi-tenant architecture (multiple organizations per user)
- Organization membership management
- Role-based access control (owner, admin, member)
- Organization-scoped resources (projects, data, etc.)
- Team collaboration
- Member invitations and management

## Prerequisites

- Go 1.21 or higher
- PostgreSQL database
- Aegis CLI tool (for migrations)

## Setup

### 1. Install Dependencies

```bash
go mod init aegis-organizations-example
go get github.com/theinventorylib/aegis
go get github.com/go-chi/chi/v5
go get github.com/lib/pq
```

### 2. Create Database

```bash
createdb aegis_orgs
```

### 3. Export and Run Migrations

```bash
# Install Aegis CLI
go install github.com/theinventorylib/aegis/cmd/aegis@latest

# Export migrations with organizations plugin
aegis export --dialect postgres --plugins organizations --output ./migrations

# Run migrations
psql aegis_orgs < migrations/001_aegis_auth_schema.sql
psql aegis_orgs < migrations/002_organizations_schema.sql
```

### 4. Configure Application

Update the database connection in `main.go`:

```go
db, err := sql.Open("postgres", "postgres://user:password@localhost/aegis_orgs?sslmode=disable")
```

Generate a secure secret:

```bash
openssl rand -base64 32
```

### 5. Run the Application

```bash
go run main.go
```

Visit http://localhost:8080

## Usage Guide

### 1. Register and Login

```bash
# Register a user
curl -X POST http://localhost:8080/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"SecurePass123","name":"Alice Smith"}' \
  -c cookies.txt

# Login (if already registered)
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"SecurePass123"}' \
  -c cookies.txt
```

### 2. Create an Organization

```bash
curl -X POST http://localhost:8080/auth/organizations \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "name": "Acme Corporation",
    "slug": "acme"
  }'
```

Response:
```json
{
  "success": true,
  "data": {
    "id": "org_01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "name": "Acme Corporation",
    "slug": "acme",
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

### 3. List Your Organizations

```bash
curl http://localhost:8080/auth/organizations -b cookies.txt
```

### 4. Get Organization Details

```bash
curl http://localhost:8080/auth/organizations/org_01ARZ3NDEKTSV4RRFFQ69G5FAV -b cookies.txt
```

### 5. Invite Team Members

First, the new member must have a user account. Then:

```bash
curl -X POST http://localhost:8080/auth/organizations/org_01ARZ3NDEKTSV4RRFFQ69G5FAV/members \
  -H "Content-Type: application/json" \
  -H "X-Organization-ID: org_01ARZ3NDEKTSV4RRFFQ69G5FAV" \
  -b cookies.txt \
  -d '{
    "user_id": "user_xxxxx",
    "role": "member"
  }'
```

Roles:
- `owner` - Full control, can delete organization
- `admin` - Can manage members and settings
- `member` - Regular access

### 6. List Organization Members

```bash
curl http://localhost:8080/auth/organizations/org_01ARZ3NDEKTSV4RRFFQ69G5FAV/members -b cookies.txt
```

### 7. Update Member Role

```bash
curl -X PATCH http://localhost:8080/auth/organizations/org_01ARZ3NDEKTSV4RRFFQ69G5FAV/members/user_xxxxx \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"role":"admin"}'
```

### 8. Remove Member

```bash
curl -X DELETE http://localhost:8080/auth/organizations/org_01ARZ3NDEKTSV4RRFFQ69G5FAV/members/user_xxxxx \
  -b cookies.txt
```

### 9. Create Organization-Scoped Resources

```bash
# Create a project (requires organization context)
curl -X POST http://localhost:8080/api/projects \
  -H "Content-Type: application/json" \
  -H "X-Organization-ID: org_01ARZ3NDEKTSV4RRFFQ69G5FAV" \
  -b cookies.txt \
  -d '{
    "name": "Website Redesign",
    "description": "Q1 2024 website redesign project"
  }'

# List organization projects
curl http://localhost:8080/api/projects \
  -H "X-Organization-ID: org_01ARZ3NDEKTSV4RRFFQ69G5FAV" \
  -b cookies.txt
```

## Architecture

### Organization Context

The organizations plugin provides middleware that extracts the organization ID from:

1. `X-Organization-ID` header (preferred for APIs)
2. Query parameter `?org=xxx`
3. Cookie `organization_id`

The middleware validates that the authenticated user is a member of the organization before proceeding.

### Data Isolation

All organization-scoped resources should:

1. Include an `organization_id` foreign key
2. Use the organization context from the request
3. Filter queries by organization ID

Example:

```go
func listProjects(w http.ResponseWriter, r *http.Request) {
    orgID := organizations.GetOrganizationID(r.Context())
    
    // Query only projects belonging to this organization
    rows, err := db.Query(`
        SELECT id, name, description 
        FROM projects 
        WHERE organization_id = $1
    `, orgID)
    // ...
}
```

### Permission Levels

The organizations plugin supports role-based access:

```go
// Require any organization membership
r.Use(orgPlugin.RequireOrganization())

// Require admin or owner role
r.Use(orgPlugin.RequireRole("admin", "owner"))

// Require owner only
r.Use(orgPlugin.RequireRole("owner"))
```

## Database Schema

The organizations plugin creates these tables:

### organizations
```sql
CREATE TABLE auth.organizations (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
```

### organization_members
```sql
CREATE TABLE auth.organization_members (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL, -- 'owner', 'admin', 'member'
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES auth.organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE,
    UNIQUE (organization_id, user_id)
);
```

## Common Patterns

### Creating Organization-Scoped Tables

```sql
CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES auth.organizations(id) ON DELETE CASCADE
);

-- Index for efficient queries
CREATE INDEX idx_projects_organization_id ON projects(organization_id);
```

### Checking User Permissions

```go
import "github.com/theinventorylib/aegis/plugins/organizations"

func handler(w http.ResponseWriter, r *http.Request) {
    orgID := organizations.GetOrganizationID(r.Context())
    role := organizations.GetOrganizationRole(r.Context())
    
    if role != "owner" && role != "admin" {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }
    
    // Proceed with admin-level operation
}
```

### Multi-Organization Users

Users can belong to multiple organizations. List all organizations for a user:

```bash
curl http://localhost:8080/auth/organizations -b cookies.txt
```

Switch between organizations by changing the `X-Organization-ID` header.

## API Endpoints Reference

### Organizations

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/auth/organizations` | Create organization |
| GET | `/auth/organizations` | List user's organizations |
| GET | `/auth/organizations/:id` | Get organization details |
| PATCH | `/auth/organizations/:id` | Update organization |
| DELETE | `/auth/organizations/:id` | Delete organization (owner only) |

### Members

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/auth/organizations/:id/members` | Add member |
| GET | `/auth/organizations/:id/members` | List members |
| PATCH | `/auth/organizations/:id/members/:userId` | Update member role |
| DELETE | `/auth/organizations/:id/members/:userId` | Remove member |

## Production Considerations

1. **Invitation System**: Implement email invitations instead of direct user_id addition
2. **Billing**: Track organization-level subscriptions and usage
3. **Resource Limits**: Enforce per-organization quotas
4. **Audit Logs**: Track organization-level events
5. **Data Export**: Allow organization owners to export their data
6. **SSO**: Consider adding SAML/SSO for enterprise organizations

## Troubleshooting

### "Organization required" Error

Make sure to include the `X-Organization-ID` header in organization-scoped requests.

### "Forbidden" Error

Verify that:
1. User is authenticated
2. User is a member of the organization
3. User has the required role (admin/owner) for the operation

### Member Not Found

When adding members, ensure:
1. The user account exists (they must sign up first)
2. You're using the correct user ID
3. The user isn't already a member

## Next Steps

- See example `04-api-jwt` for stateless JWT authentication
- Combine with OAuth example for social login in multi-tenant apps
- Add billing integration (Stripe, etc.) at the organization level
- Implement invitation emails with verification tokens

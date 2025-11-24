# Organizations Plugin

The Organizations plugin provides multi-tenant organization and team management capabilities.

## Installation

```bash
go get github.com/theinventorylib/aegis/plugins/organizations
```

## Configuration

```go
import "github.com/theinventorylib/aegis/plugins/organizations"

orgPlugin := organizations.New(dbProvider)
```

## Endpoints

All endpoints require a valid session cookie or Bearer token.

### Organizations
- `POST /organizations`: Create a new organization.
- `GET /organizations`: List organizations the user belongs to.
- `GET /organizations/:id`: Get details for a specific organization.
- `PUT /organizations/:id`: Update organization details.
- `DELETE /organizations/:id`: Delete an organization.

### Organization Members
- `POST /organizations/:id/members`: Add a member to an organization.
- `GET /organizations/:id/members`: List members of an organization.
- `PATCH /organizations/:id/members/:userId`: Update a member's role.
- `DELETE /organizations/:id/members/:userId`: Remove a member.

### Teams
- `POST /organizations/:id/teams`: Create a team within an organization.
- `GET /organizations/:id/teams`: List teams in an organization.
- `GET /teams/:teamId`: Get details for a specific team.
- `PUT /teams/:teamId`: Update team details.
- `DELETE /teams/:teamId`: Delete a team.

### Team Members
- `POST /teams/:teamId/members`: Add a member to a team.
- `GET /teams/:teamId/members`: List members of a team.
- `PATCH /teams/:teamId/members/:userId`: Update a team member's role.
- `DELETE /teams/:teamId/members/:userId`: Remove a member from a team.

// Package main demonstrates multi-tenant organization support with Aegis.
//
// This example shows:
//   - Creating and managing organizations
//   - Organization membership and roles
//   - Tenant isolation (users can only access their organization's data)
//   - Team collaboration features
//   - Organization-scoped API endpoints
//
// Run this example:
//  1. Set up a PostgreSQL database
//  2. Run migrations with organizations plugin
//  3. Update configuration below
//  4. go run main.go
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	"github.com/theinventorylib/aegis"
	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/plugins/organizations"
	"github.com/theinventorylib/aegis/router"
)

func main() {
	// 1. Connect to database
	db, err := sql.Open("postgres", "postgres://user:password@localhost/aegis_orgs?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Database ping failed:", err)
	}

	// 2. Create HTTP router with wrapper
	mux := chi.NewRouter()
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	r := router.NewChiRouter(mux)

	// 3. Create organizations plugin
	orgPlugin := organizations.New(nil, plugins.DialectPostgres)

	cfg := config.Default().WithDB(db).WithRouter(r).WithSecret([]byte("your-32-byte-secret-key-here!!!!"))
	a, err := aegis.New(context.Background(), cfg)
	if err != nil {
		log.Fatal("Failed to create Aegis instance:", err)
	}

	// Register organizations plugin
	if err := a.Use(context.Background(), orgPlugin); err != nil {
		log.Fatal("Failed to register organizations plugin:", err)
	}

	// 4. Mount Aegis routes
	// Organizations plugin adds these routes:
	//   - POST   /auth/organizations                - Create organization
	//   - GET    /auth/organizations                - List user's organizations
	//   - GET    /auth/organizations/:id            - Get organization details
	//   - PATCH  /auth/organizations/:id            - Update organization
	//   - DELETE /auth/organizations/:id            - Delete organization
	//   - POST   /auth/organizations/:id/members    - Invite member
	//   - GET    /auth/organizations/:id/members    - List members
	//   - DELETE /auth/organizations/:id/members/:userId - Remove member
	a.MountRoutes("/auth")

	// 5. Public routes
	mux.Get("/", homeHandler)

	// 6. Protected routes (require authentication)
	mux.Group(func(r chi.Router) {
		r.Use(a.RequireAuth())

		r.Get("/dashboard", dashboardHandler)

		// Organization-scoped routes (require organization context)
		r.Group(func(r chi.Router) {
			r.Use(orgPlugin.RequireOrganizationMemberMiddleware()) // Require organization membership

			r.Get("/api/projects", listProjectsHandler)
			r.Post("/api/projects", createProjectHandler)
			r.Get("/api/projects/{id}", getProjectHandler)
			r.Delete("/api/projects/{id}", deleteProjectHandler)

			r.Get("/api/team", teamMembersHandler)
		})
	})

	log.Println("Server starting on http://localhost:8080")
	log.Println("Organizations plugin enabled")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Aegis Organizations Example</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 900px; margin: 50px auto; padding: 20px; }
        h1 { color: #333; }
        .section { margin: 30px 0; padding: 20px; background: #f9f9f9; border-radius: 4px; }
        code { background: #e8e8e8; padding: 2px 6px; border-radius: 3px; font-size: 14px; }
        pre { background: #f4f4f4; padding: 15px; border-radius: 4px; overflow-x: auto; }
        .endpoint { margin: 10px 0; }
        .method { font-weight: bold; color: #2c5aa0; }
    </style>
</head>
<body>
    <h1>🏢 Aegis Organizations Example</h1>
    <p>This example demonstrates multi-tenant organization management with Aegis.</p>
    
    <div class="section">
        <h2>Features</h2>
        <ul>
            <li><strong>Multi-tenancy:</strong> Each user can belong to multiple organizations</li>
            <li><strong>Role-based access:</strong> Owner, admin, and member roles</li>
            <li><strong>Team collaboration:</strong> Invite and manage team members</li>
            <li><strong>Data isolation:</strong> Organization-scoped data access</li>
            <li><strong>Resource management:</strong> Projects, documents, etc. scoped to organizations</li>
        </ul>
    </div>
    
    <div class="section">
        <h2>Workflow Example</h2>
        <ol>
            <li>User signs up and logs in</li>
            <li>User creates an organization</li>
            <li>User invites team members</li>
            <li>Team members create organization-scoped projects</li>
            <li>Only organization members can access the projects</li>
        </ol>
    </div>
    
    <div class="section">
        <h2>API Endpoints</h2>
        
        <h3>Organizations</h3>
        <div class="endpoint"><span class="method">POST</span> <code>/auth/organizations</code> - Create organization</div>
        <div class="endpoint"><span class="method">GET</span> <code>/auth/organizations</code> - List user's organizations</div>
        <div class="endpoint"><span class="method">GET</span> <code>/auth/organizations/:id</code> - Get organization</div>
        <div class="endpoint"><span class="method">PATCH</span> <code>/auth/organizations/:id</code> - Update organization</div>
        <div class="endpoint"><span class="method">DELETE</span> <code>/auth/organizations/:id</code> - Delete organization</div>
        
        <h3>Members</h3>
        <div class="endpoint"><span class="method">POST</span> <code>/auth/organizations/:id/members</code> - Invite member</div>
        <div class="endpoint"><span class="method">GET</span> <code>/auth/organizations/:id/members</code> - List members</div>
        <div class="endpoint"><span class="method">PATCH</span> <code>/auth/organizations/:id/members/:userId</code> - Update member role</div>
        <div class="endpoint"><span class="method">DELETE</span> <code>/auth/organizations/:id/members/:userId</code> - Remove member</div>
        
        <h3>Projects (Organization-scoped)</h3>
        <div class="endpoint"><span class="method">GET</span> <code>/api/projects</code> - List organization projects</div>
        <div class="endpoint"><span class="method">POST</span> <code>/api/projects</code> - Create project</div>
        <div class="endpoint"><span class="method">GET</span> <code>/api/projects/:id</code> - Get project</div>
        <div class="endpoint"><span class="method">DELETE</span> <code>/api/projects/:id</code> - Delete project</div>
    </div>
    
    <div class="section">
        <h2>cURL Examples</h2>
        <pre>
# 1. Sign up
curl -X POST http://localhost:8080/auth/signup \\
  -H "Content-Type: application/json" \\
  -d '{"email":"alice@example.com","password":"SecurePass123","name":"Alice"}' \\
  -c cookies.txt

# 2. Create an organization
curl -X POST http://localhost:8080/auth/organizations \\
  -H "Content-Type: application/json" \\
  -b cookies.txt \\
  -d '{"name":"Acme Corp","slug":"acme"}'

# 3. List organizations
curl http://localhost:8080/auth/organizations -b cookies.txt

# 4. Set organization header for subsequent requests
ORG_ID="org_xxx" # Replace with actual ID from step 2

# 5. Create a project (organization-scoped)
curl -X POST http://localhost:8080/api/projects \\
  -H "Content-Type: application/json" \\
  -H "X-Organization-ID: $ORG_ID" \\
  -b cookies.txt \\
  -d '{"name":"My Project","description":"A sample project"}'

# 6. List projects
curl http://localhost:8080/api/projects \\
  -H "X-Organization-ID: $ORG_ID" \\
  -b cookies.txt

# 7. Invite a team member
curl -X POST http://localhost:8080/auth/organizations/$ORG_ID/members \\
  -H "Content-Type: application/json" \\
  -b cookies.txt \\
  -d '{"user_id":"user_xxx","role":"member"}'
        </pre>
    </div>
</body>
</html>
	`))
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil || user == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        .card { background: #f9f9f9; padding: 20px; margin: 20px 0; border-radius: 4px; }
        button { padding: 10px 20px; background: #2c5aa0; color: white; border: none; border-radius: 4px; cursor: pointer; }
        button:hover { background: #1e4080; }
    </style>
</head>
<body>
    <h1>Dashboard</h1>
    
    <div class="card">
        <h2>Welcome, %s!</h2>
        <p>Email: %s</p>
        <p>User ID: %s</p>
    </div>
    
    <div class="card">
        <h3>Your Organizations</h3>
        <div id="organizations">Loading...</div>
        <button onclick="createOrg()">Create Organization</button>
    </div>
    
    <script>
        async function loadOrganizations() {
            const res = await fetch('/auth/organizations');
            const data = await res.json();
            const div = document.getElementById('organizations');
            
            if (data.data && data.data.length > 0) {
                div.innerHTML = '<ul>' + data.data.map(org => 
                    '<li><strong>' + org.name + '</strong> (' + org.role + ') - ID: ' + org.id + '</li>'
                ).join('') + '</ul>';
            } else {
                div.innerHTML = '<p>No organizations yet. Create one to get started!</p>';
            }
        }
        
        async function createOrg() {
            const name = prompt('Organization name:');
            const slug = prompt('Organization slug (URL-friendly):');
            
            if (name && slug) {
                const res = await fetch('/auth/organizations', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name, slug })
                });
                
                if (res.ok) {
                    alert('Organization created!');
                    loadOrganizations();
                } else {
                    const error = await res.json();
                    alert('Error: ' + (error.error || 'Unknown error'));
                }
            }
        }
        
        loadOrganizations();
    </script>
</body>
</html>
	`, user.Name, user.Email, user.ID)))
}

// Mock project storage (in real app, use database with organization_id foreign key)
var projects = make(map[string][]Project)

type Project struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	CreatedBy      string `json:"created_by"`
}

func listProjectsHandler(w http.ResponseWriter, r *http.Request) {
	// Get organization ID from path param
	orgID := core.GetSanitizedPathParam(r, "id")
	if orgID == "" {
		http.Error(w, "Organization required", http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"success":         true,
		"projects":        projects[orgID],
		"organization_id": orgID,
	})
}

func createProjectHandler(w http.ResponseWriter, r *http.Request) {
	// Get organization ID from path param
	orgID := core.GetSanitizedPathParam(r, "id")
	if orgID == "" {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return
	}

	user, err := core.GetUser(r.Context())
	if err != nil || user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Sanitize inputs
	input.Name = core.SanitizeString(input.Name, nil)
	input.Description = core.SanitizeMultiline(input.Description, 500)

	project := Project{
		ID:             fmt.Sprintf("proj_%d", len(projects[orgID])+1),
		OrganizationID: orgID,
		Name:           input.Name,
		Description:    input.Description,
		CreatedBy:      user.ID,
	}

	if projects[orgID] == nil {
		projects[orgID] = []Project{}
	}
	projects[orgID] = append(projects[orgID], project)

	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"project": project,
	})
}

func getProjectHandler(w http.ResponseWriter, r *http.Request) {
	// Get organization ID from path param
	orgID := core.GetSanitizedPathParam(r, "id")
	if orgID == "" {
		http.Error(w, "Organization required", http.StatusBadRequest)
		return
	}

	projectID := core.GetSanitizedPathParam(r, "id")

	for _, p := range projects[orgID] {
		if p.ID == projectID {
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"project": p,
			})
			return
		}
	}

	http.Error(w, "Project not found", http.StatusNotFound)
}

func deleteProjectHandler(w http.ResponseWriter, r *http.Request) {
	// Get organization ID from path param
	orgID := core.GetSanitizedPathParam(r, "id")
	if orgID == "" {
		http.Error(w, "Organization required", http.StatusBadRequest)
		return
	}

	projectID := core.GetSanitizedPathParam(r, "id")

	for i, p := range projects[orgID] {
		if p.ID == projectID {
			projects[orgID] = append(projects[orgID][:i], projects[orgID][i+1:]...)
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"message": "Project deleted",
			})
			return
		}
	}

	http.Error(w, "Project not found", http.StatusNotFound)
}

func teamMembersHandler(w http.ResponseWriter, r *http.Request) {
	// Get organization ID from path param
	orgID := core.GetSanitizedPathParam(r, "id")
	if orgID == "" {
		http.Error(w, "Organization required", http.StatusBadRequest)
		return
	}

	// In real app, query members from database
	json.NewEncoder(w).Encode(map[string]any{
		"success":         true,
		"message":         "Use GET /auth/organizations/" + orgID + "/members to see team members",
		"organization_id": orgID,
	})
}

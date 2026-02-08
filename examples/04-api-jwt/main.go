// Package main demonstrates stateless JWT API authentication with Aegis.
//
// This example shows:
//   - API-only mode (no cookies, CSRF disabled)
//   - JWT token generation and validation
//   - Stateless authentication (no session storage)
//   - Token refresh mechanism
//   - Bearer token authentication (auto-enabled in API mode)
//
// Run this example:
//  1. Set up a PostgreSQL database
//  2. Run migrations with JWT plugin
//  3. Update configuration below
//  4. go run main.go
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	_ "github.com/lib/pq"
	"github.com/theinventorylib/aegis"
	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/plugins/jwt"
	"github.com/theinventorylib/aegis/router"
)

func main() {
	// 1. Connect to database
	db, err := sql.Open("postgres", "postgres://user:password@localhost/aegis_jwt?sslmode=disable")
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
	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)

	// 3. Configure CORS for API access
	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // In production, specify actual origins
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false, // No cookies in API mode
		MaxAge:           300,
	}))

	r := router.NewChiRouter(mux)

	// 4. Create JWT plugin
	jwtConfig := &jwt.Config{
		Issuer:             "aegis-jwt-example",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	jwtPlugin := jwt.New(jwtConfig, nil, plugins.DialectPostgres)

	// 5. Create Aegis instance in API-only mode
	// Bearer token auth is auto-enabled via WithAPIOnlyMode
	a, err := aegis.New(context.Background(),
		config.WithDB(db),
		config.WithRouter(r),
		config.WithSecret([]byte("your-32-byte-secret-key-here!!!!")),
		config.WithAPIOnlyMode(true), // Disable cookies/CSRF, auto-enable Bearer auth
	)
	if err != nil {
		log.Fatal("Failed to create Aegis instance:", err)
	}

	// Register plugins
	if err := a.Use(context.Background(), jwtPlugin); err != nil {
		log.Fatal("Failed to register JWT plugin:", err)
	}

	// 6. Mount Aegis routes
	// JWT plugin adds these routes:
	//   - POST /auth/jwt/token         - Generate JWT from session
	//   - POST /auth/jwt/refresh       - Refresh JWT token
	//   - POST /auth/jwt/revoke        - Revoke JWT token
	a.MountRoutes("/auth")

	// 6. Public endpoints
	mux.Get("/", apiDocsHandler)
	mux.Get("/health", healthHandler)

	// 7. Protected endpoints (JWT or Bearer token required)
	mux.Group(func(r chi.Router) {
		r.Use(a.RequireAuth())

		r.Get("/api/profile", profileHandler)
		r.Get("/api/data", dataHandler)
		r.Post("/api/data", createDataHandler)
		r.Delete("/api/data/{id}", deleteDataHandler)
	})

	// 8. Admin endpoints (example of role-based access)
	mux.Group(func(r chi.Router) {
		r.Use(a.RequireAuth())
		// In production, add role checking middleware here

		r.Get("/api/admin/users", adminUsersHandler)
		r.Get("/api/admin/stats", adminStatsHandler)
	})

	log.Println("🚀 JWT API Server starting on http://localhost:8080")
	log.Println("📖 API documentation: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func apiDocsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Aegis JWT API Example</title>
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; max-width: 1000px; margin: 50px auto; padding: 20px; background: #f5f5f5; }
        .container { background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 3px solid #2c5aa0; padding-bottom: 10px; }
        h2 { color: #2c5aa0; margin-top: 30px; }
        .endpoint { background: #f9f9f9; padding: 15px; margin: 15px 0; border-radius: 4px; border-left: 4px solid #2c5aa0; }
        .method { display: inline-block; padding: 4px 12px; border-radius: 3px; font-weight: bold; color: white; margin-right: 10px; }
        .post { background: #28a745; }
        .get { background: #007bff; }
        .delete { background: #dc3545; }
        code { background: #e8e8e8; padding: 2px 8px; border-radius: 3px; font-family: 'Courier New', monospace; font-size: 14px; }
        pre { background: #2d2d2d; color: #f8f8f8; padding: 20px; border-radius: 4px; overflow-x: auto; }
        .section { margin: 30px 0; }
        .feature { display: inline-block; margin: 10px 20px 10px 0; padding: 8px 15px; background: #e3f2fd; border-radius: 20px; color: #1976d2; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔐 Aegis JWT API Example</h1>
        <p>This is a stateless REST API using JWT tokens for authentication. No cookies, no sessions stored server-side.</p>
        
        <div class="section">
            <h2>✨ Features</h2>
            <span class="feature">Stateless Authentication</span>
            <span class="feature">JWT Tokens</span>
            <span class="feature">API Keys (Bearer)</span>
            <span class="feature">Token Refresh</span>
            <span class="feature">CORS Enabled</span>
            <span class="feature">No Cookies</span>
        </div>
        
        <div class="section">
            <h2>🚀 Quick Start</h2>
            <pre># 1. Register
curl -X POST http://localhost:8080/auth/signup \\
  -H "Content-Type: application/json" \\
  -d '{"email":"user@example.com","password":"SecurePass123","name":"John Doe"}'

# 2. Login (get JWT token)
curl -X POST http://localhost:8080/auth/login \\
  -H "Content-Type: application/json" \\
  -d '{"email":"user@example.com","password":"SecurePass123"}'

# Returns: {"success":true,"token":"eyJhbGc...","refresh_token":"..."}

# 3. Use token in subsequent requests
TOKEN="eyJhbGc..."
curl http://localhost:8080/api/profile \\
  -H "Authorization: Bearer $TOKEN"

# 4. Refresh token when it expires
curl -X POST http://localhost:8080/auth/jwt/refresh \\
  -H "Content-Type: application/json" \\
  -d '{"refresh_token":"your-refresh-token"}'</pre>
        </div>
        
        <div class="section">
            <h2>📡 Authentication Endpoints</h2>
            
            <div class="endpoint">
                <span class="method post">POST</span> <code>/auth/signup</code>
                <p>Register a new user account</p>
                <p><strong>Body:</strong> <code>{"email": "user@example.com", "password": "SecurePass123", "name": "John Doe"}</code></p>
                <p><strong>Returns:</strong> User object + JWT token</p>
            </div>
            
            <div class="endpoint">
                <span class="method post">POST</span> <code>/auth/login</code>
                <p>Login with email and password</p>
                <p><strong>Body:</strong> <code>{"email": "user@example.com", "password": "SecurePass123"}</code></p>
                <p><strong>Returns:</strong> <code>{"success": true, "token": "JWT...", "refresh_token": "..."}</code></p>
            </div>
            
            <div class="endpoint">
                <span class="method post">POST</span> <code>/auth/jwt/refresh</code>
                <p>Refresh an expired access token</p>
                <p><strong>Body:</strong> <code>{"refresh_token": "your-refresh-token"}</code></p>
                <p><strong>Returns:</strong> New access token and refresh token</p>
            </div>
            
            <div class="endpoint">
                <span class="method post">POST</span> <code>/auth/jwt/revoke</code>
                <p>Revoke a JWT token (blacklist it)</p>
                <p><strong>Headers:</strong> <code>Authorization: Bearer {token}</code></p>
            </div>
        </div>
        

        
        <div class="section">
            <h2>📊 Protected API Endpoints</h2>
            
            <div class="endpoint">
                <span class="method get">GET</span> <code>/api/profile</code>
                <p>Get current user profile</p>
                <p><strong>Headers:</strong> <code>Authorization: Bearer {token}</code></p>
            </div>
            
            <div class="endpoint">
                <span class="method get">GET</span> <code>/api/data</code>
                <p>Get user's data</p>
            </div>
            
            <div class="endpoint">
                <span class="method post">POST</span> <code>/api/data</code>
                <p>Create new data entry</p>
            </div>
            
            <div class="endpoint">
                <span class="method delete">DELETE</span> <code>/api/data/:id</code>
                <p>Delete data entry</p>
            </div>
        </div>
        
        <div class="section">
            <h2>🔒 Security Features</h2>
            <ul>
                <li><strong>Stateless:</strong> No server-side session storage, JWT contains all necessary info</li>
                <li><strong>Short-lived tokens:</strong> Access tokens expire in 15 minutes</li>
                <li><strong>Refresh tokens:</strong> Long-lived (7 days) for obtaining new access tokens</li>
                <li><strong>Token revocation:</strong> Blacklist tokens before expiry</li>
                <li><strong>Bearer auth:</strong> Auto-enabled in API mode via config</li>
                <li><strong>CORS enabled:</strong> Safe cross-origin requests</li>
            </ul>
        </div>
        
        <div class="section">
            <h2>💡 Usage Tips</h2>
            <ul>
                <li>Store tokens securely (never in localStorage for web apps)</li>
                <li>Implement token refresh logic before expiry</li>
                <li>Bearer auth is auto-enabled in API mode (no plugin needed)</li>
                <li>Set short expiry times for access tokens (5-15 minutes)</li>
                <li>Always use HTTPS in production</li>
                <li>Implement rate limiting for public endpoints</li>
            </ul>
        </div>
    </div>
</body>
</html>
	`))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "healthy",
		"service": "aegis-jwt-api",
		"time":    time.Now().Format(time.RFC3339),
	})
}

func profileHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil || user == nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"user": map[string]any{
			"id":         user.ID,
			"email":      user.Email,
			"name":       user.Name,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		},
	})
}

func dataHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil || user == nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Mock data
	data := []map[string]any{
		{"id": "1", "title": "Item 1", "user_id": user.ID},
		{"id": "2", "title": "Item 2", "user_id": user.ID},
		{"id": "3", "title": "Item 3", "user_id": user.ID},
	}

	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"data":    data,
		"count":   len(data),
	})
}

func createDataHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil || user == nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var input map[string]any
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Mock creation
	result := map[string]any{
		"id":         "new_id",
		"user_id":    user.ID,
		"created_at": time.Now(),
	}
	for k, v := range input {
		result[k] = v
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"data":    result,
	})
}

func deleteDataHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil || user == nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := chi.URLParam(r, "id")

	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "Deleted item " + id,
	})
}

func adminUsersHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil || user == nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// In production, check if user has admin role

	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"users": []map[string]any{
			{"id": "1", "email": "user1@example.com", "name": "User 1"},
			{"id": "2", "email": "user2@example.com", "name": "User 2"},
		},
	})
}

func adminStatsHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil || user == nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"stats": map[string]any{
			"total_users":    1234,
			"active_users":   456,
			"total_requests": 98765,
		},
	})
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error":   message,
	})
}

// extractToken extracts JWT from Authorization header
func extractToken(r *http.Request) string {
	bearerToken := r.Header.Get("Authorization")
	if len(bearerToken) > 7 && strings.ToLower(bearerToken[:7]) == "bearer " {
		return bearerToken[7:]
	}
	return ""
}

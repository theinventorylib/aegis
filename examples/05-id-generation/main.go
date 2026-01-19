// Package main demonstrates ID generation strategies in Aegis.
//
// This example shows:
//   - Configuring different ID generation strategies (ULID, UUID, Custom)
//   - How ID generation affects user and session IDs
//   - Demonstrating ID generation through API endpoints
//
// Run this example:
//  1. Set up a PostgreSQL database
//  2. Run migrations: aegis export --dialect postgres --output ./migrations
//  3. Update the database connection string below
//  4. go run main.go
//  5. Visit http://localhost:8080
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	"github.com/theinventorylib/aegis"
	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/router"
)

func main() {
	// 1. Connect to database
	db, err := sql.Open("postgres", "postgres://user:password@localhost/aegis_id_gen?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// 2. Create HTTP router with chi
	mux := chi.NewRouter()
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	r := router.NewChiRouter(mux)

	// 3. Create Aegis instance with ULID strategy (default)
	a, err := aegis.New(context.Background(),
		config.WithDB(db),
		config.WithRouter(r),
		config.WithSecret([]byte("your-32-byte-secret-key-here!!!!")),
		config.WithIDStrategy(core.IDStrategyULID), // Explicitly set ULID
	)
	if err != nil {
		log.Fatal("Failed to create Aegis instance:", err)
	}

	// 4. Mount Aegis routes
	a.MountRoutes("/auth")

	// 5. Add custom routes to demonstrate ID generation
	mux.Get("/", homeHandler)
	mux.Get("/ids", idsHandler)
	mux.Get("/strategy", strategyHandler)

	// 6. Protected routes
	mux.Group(func(r chi.Router) {
		r.Use(a.RequireAuth())
		r.Get("/dashboard", dashboardHandler)
	})

	log.Println("Server starting on http://localhost:8080")
	log.Println("ID Strategy:", core.GetIDStrategy())
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Aegis ID Generation Example</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        h1 { color: #333; }
        .button { 
            display: inline-block; 
            padding: 12px 24px; 
            margin: 10px 5px; 
            background: #2c5aa0; 
            color: white; 
            text-decoration: none; 
            border-radius: 4px;
        }
        .button:hover { background: #1e4080; }
        .section { margin: 30px 0; padding: 20px; background: #f9f9f9; border-radius: 4px; }
        code { background: #e8e8e8; padding: 2px 6px; border-radius: 3px; }
        .id-list { background: #fff; padding: 10px; border: 1px solid #ddd; border-radius: 4px; margin: 10px 0; }
    </style>
</head>
<body>
    <h1>Aegis ID Generation Example</h1>
    <p>This example demonstrates different ID generation strategies in Aegis.</p>
    
    <div class="section">
        <h2>Current Strategy</h2>
        <p>Current ID Strategy: <code>` + string(core.GetIDStrategy()) + `</code></p>
        <a href="/strategy" class="button">View Strategy Info</a>
    </div>
    
    <div class="section">
        <h2>Generate IDs</h2>
        <a href="/ids" class="button">Generate Sample IDs</a>
    </div>
    
    <div class="section">
        <h2>Authentication</h2>
        <p>Try creating a user account to see how IDs are used:</p>
        <a href="/auth/signup" class="button">Sign Up</a>
        <a href="/auth/login" class="button">Login</a>
    </div>
</body>
</html>
`))
}

func idsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ids := make([]string, 5)
	for i := 0; i < 5; i++ {
		ids[i] = core.GenerateID()
	}

	response := map[string]any{
		"strategy": core.GetIDStrategy(),
		"ids":      ids,
		"length":   len(ids[0]),
	}

	json.NewEncoder(w).Encode(response)
}

func strategyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	strategies := []map[string]any{
		{
			"name":        "ULID",
			"description": "Universally Unique Lexicographically Sortable Identifier",
			"length":      26,
			"sortable":    true,
			"time_based":  true,
		},
		{
			"name":        "UUID",
			"description": "Universally Unique Identifier (v4)",
			"length":      36,
			"sortable":    false,
			"time_based":  false,
		},
	}

	response := map[string]any{
		"current_strategy":     core.GetIDStrategy(),
		"available_strategies": strategies,
	}

	json.NewEncoder(w).Encode(response)
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Dashboard - Aegis ID Generation</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        .button { 
            display: inline-block; 
            padding: 12px 24px; 
            margin: 10px 5px; 
            background: #2c5aa0; 
            color: white; 
            text-decoration: none; 
            border-radius: 4px;
        }
        .button:hover { background: #1e4080; }
    </style>
</head>
<body>
    <h1>Dashboard</h1>
    <p>You are authenticated! Your user ID uses the current ID strategy.</p>
    <a href="/" class="button">Back to Home</a>
    <a href="/auth/logout" class="button">Logout</a>
</body>
</html>
`))
}

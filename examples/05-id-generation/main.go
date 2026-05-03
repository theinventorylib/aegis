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
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	"github.com/theinventorylib/aegis"
	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/router/routers"
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
	r := routers.NewChiRouter(mux)

	// 3. Create Aegis instance with ULID strategy (default)
	cfg := config.Default().
		WithDB(db).
		WithRouter(r).
		WithSecret([]byte("your-32-byte-secret-key-here!!!!")).
		WithIDStrategy(core.IDStrategyULID) // Explicitly set ULID

	a, err := aegis.New(context.Background(), cfg)
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
        <a href="/auth/default/signup" class="button">Sign Up</a>
        <a href="/auth/default/login" class="button">Login</a>
    </div>
</body>
</html>
`))
}

// Response types for consistent API responses

type IDListResponse struct {
	Strategy string   `json:"strategy"`
	IDs      []string `json:"ids"`
	Length   int      `json:"length"`
}

type IDStrategyInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Length      int    `json:"length"`
	Sortable    bool   `json:"sortable"`
	TimeBased   bool   `json:"time_based"`
}

type StrategyResponse struct {
	CurrentStrategy     string           `json:"current_strategy"`
	AvailableStrategies []IDStrategyInfo `json:"available_strategies"`
}

func idsHandler(w http.ResponseWriter, r *http.Request) {
	ids := make([]string, 5)
	for i := 0; i < 5; i++ {
		ids[i] = core.GenerateID()
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Data: IDListResponse{
			Strategy: core.GetIDStrategy(),
			IDs:      ids,
			Length:   len(ids[0]),
		},
	})
}

func strategyHandler(w http.ResponseWriter, r *http.Request) {
	strategies := []IDStrategyInfo{
		{
			Name:        "ULID",
			Description: "Universally Unique Lexicographically Sortable Identifier",
			Length:      26,
			Sortable:    true,
			TimeBased:   true,
		},
		{
			Name:        "UUID",
			Description: "Universally Unique Identifier (v4)",
			Length:      36,
			Sortable:    false,
			TimeBased:   false,
		},
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Data: StrategyResponse{
			CurrentStrategy:     core.GetIDStrategy(),
			AvailableStrategies: strategies,
		},
	})
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
    <a href="/auth/default/logout" class="button">Logout</a>
</body>
</html>
`))
}

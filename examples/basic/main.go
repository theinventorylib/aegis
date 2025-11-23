package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/theinventorylib/aegis"
	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/db"
	"github.com/theinventorylib/aegis/server"

	// Import PostgreSQL driver (users choose their preferred driver)
	_ "github.com/lib/pq"
)

func main() {
	// Initialize database connection using standard database/sql
	// Users choose their own driver - here we use lib/pq for PostgreSQL
	connString := "postgres://user:password@localhost:5432/aegis_db?sslmode=disable"

	// Standard Go database setup - works with any SQL driver
	sqlDB, err := sql.Open("postgres", connString)
	if err != nil {
		log.Fatal("Failed to open database connection:", err)
	}
	defer sqlDB.Close()

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Create Chi router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Create Aegis router adapter
	aegisRouter := server.NewChiRouter(r)

	// Initialize Aegis with database/sql connection
	// Specify the dialect so Aegis knows the SQL syntax to use
	auth, err := aegis.New(
		config.WithDB(sqlDB, db.PostgreSQL),
		config.WithRouter(aegisRouter),
		config.WithJWTSecret([]byte("your-secret-key-change-in-production")),
		config.WithCSRFSecret([]byte("your-csrf-secret-change-in-production")),
		config.WithCookieSecure(false), // Set to true in production with HTTPS
	)
	if err != nil {
		log.Fatal("Failed to initialize Aegis:", err)
	}

	// Mount authentication routes
	auth.MountRoutes("/auth")

	// Apply auth middleware to protected routes
	r.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware())

		// Example protected route
		r.Get("/api/protected", func(w http.ResponseWriter, r *http.Request) {
			user, err := auth.GetUser(r.Context())
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"message":"Hello, user %s!"}`, user.ID)
		})
	})

	// Start server
	port := ":3000"
	fmt.Printf("Server running on http://localhost%s\n", port)
	fmt.Println("Authentication endpoints:")

	fmt.Println("  POST   /auth/logout")
	fmt.Println("  GET    /auth/user")
	fmt.Println("  POST   /auth/otp/send")
	fmt.Println("  POST   /auth/otp/verify")

	fmt.Println()
	fmt.Println("Protected endpoints:")
	fmt.Println("  GET    /api/protected")

	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatal(err)
	}
}

// Package main demonstrates bearer token authentication with Aegis.
package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/theinventorylib/aegis"
	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/db"
	"github.com/theinventorylib/aegis/plugins/bearer"
	"github.com/theinventorylib/aegis/server"

	_ "github.com/lib/pq"
)

func main() {
	// Connect to database
	database, err := sql.Open("postgres", "postgres://postgres:postgres@localhost:5432/aegis_dev?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	// Create router
	mux := http.NewServeMux()
	router := server.NewDefaultRouter(mux)

	// Create Aegis instance
	auth, err := aegis.New(
		config.WithDB(database, db.PostgreSQL),
		config.WithRouter(router),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Register bearer plugin to enable Bearer token authentication
	// Note: Bearer auth works with core session tokens
	bearerPlugin := bearer.New(&bearer.Config{})
	if err := auth.Use(context.Background(), bearerPlugin); err != nil {
		log.Fatal(err)
	}

	// Mount routes
	auth.MountRoutes("/auth")

	// Add a protected test route
	protectedHandler := auth.RequireAuth()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := auth.GetUser(r.Context())
		if err != nil {
			http.Error(w, "Failed to get user", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Authenticated successfully","user_id":"` + user.ID + `"}`))
	}))

	router.GET("/protected", protectedHandler.ServeHTTP)

	// Start server
	log.Println("Bearer authentication is enabled!")
	log.Println("Test with:")
	log.Println("  1. Create a session (via email/password/etc plugins)")
	log.Println("  2. Use session token: curl -X GET http://localhost:8080/protected -H 'Authorization: Bearer <session_token>'")
	log.Println("  3. Or use cookie: curl -X GET http://localhost:8080/protected -H 'Cookie: aegis_session=<session_token>'")

	// Start server with timeouts
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("Server starting on :8080")
	log.Fatal(srv.ListenAndServe())
}

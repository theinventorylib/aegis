// Package main demonstrates OpenAPI documentation generation with Aegis.
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
	"github.com/theinventorylib/aegis/plugins/openapi"
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

	// Register OpenAPI plugin with custom configuration
	openapiPlugin := openapi.New(&openapi.Config{
		Title:       "Aegis Auth API Example",
		Version:     "1.0.0",
		Description: "Example API demonstrating OpenAPI plugin with custom endpoints",
		Servers: []openapi.Server{
			{
				URL:         "http://localhost:8080",
				Description: "Development server",
			},
		},
		Contact: &openapi.Contact{
			Name:  "API Support",
			Email: "support@example.com",
		},
		License: &openapi.License{
			Name: "MIT",
			URL:  "https://opensource.org/licenses/MIT",
		},
		EnableScalarUI: true,
		BasePath:       "/auth",
	})
	if err := auth.Use(context.Background(), openapiPlugin); err != nil {
		log.Fatal(err)
	}

	// Mount routes
	auth.MountRoutes("/auth")

	// Add custom endpoints and document them
	addCustomEndpoints(auth, router, openapiPlugin)

	// Start server with timeouts
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("Server starting on :8080")
	log.Println("OpenAPI Documentation:")
	log.Println("  - Scalar UI: http://localhost:8080/auth/docs")
	log.Println("  - OpenAPI JSON: http://localhost:8080/auth/openapi.json")
	log.Fatal(srv.ListenAndServe())
}

func addCustomEndpoints(auth *aegis.Aegis, router server.Router, oapi *openapi.Plugin) {
	// Add a custom tag
	oapi.RegisterTag(openapi.Tag{
		Name:        "Custom",
		Description: "Custom endpoints for demonstration",
	})

	// Add a custom schema
	oapi.RegisterSchema("CustomResponse", openapi.ObjectSchema(
		"Custom response object",
		map[string]*openapi.Schema{
			"message": openapi.StringSchema("Response message"),
			"data": openapi.ObjectSchema("", map[string]*openapi.Schema{
				"id":    openapi.UUIDSchema("Item ID"),
				"name":  openapi.StringSchema("Item name"),
				"count": openapi.IntegerSchema("Item count"),
			}, []string{"id", "name"}),
		},
		[]string{"message"},
	))

	// Add a public custom endpoint
	router.GET("/api/public/info", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Public endpoint","version":"1.0.0"}`))
	})

	// Document the public endpoint
	oapi.RegisterEndpoint("GET", "/api/public/info", &openapi.Operation{
		Tags:        []string{"Custom"},
		Summary:     "Get API information",
		Description: "Returns public API information",
		OperationID: "getApiInfo",
		Responses: map[string]*openapi.Response{
			"200": {
				Description: "API information",
				Content: map[string]openapi.MediaType{
					"application/json": {
						Schema: openapi.ObjectSchema("", map[string]*openapi.Schema{
							"message": openapi.StringSchema("Info message"),
							"version": openapi.StringSchema("API version"),
						}, []string{"message", "version"}),
					},
				},
			},
		},
	})

	// Add a protected custom endpoint
	protectedHandler := auth.RequireAuth()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := auth.GetUser(r.Context())
		if err != nil {
			http.Error(w, "Failed to get user", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Custom protected endpoint","user_id":"` + user.ID + `","data":{"id":"123","name":"Example","count":42}}`))
	}))

	router.GET("/api/custom/data", protectedHandler.ServeHTTP)

	// Document the protected endpoint
	oapi.RegisterEndpoint("GET", "/api/custom/data", &openapi.Operation{
		Tags:        []string{"Custom"},
		Summary:     "Get custom data",
		Description: "Returns custom data for authenticated user",
		OperationID: "getCustomData",
		Security: []openapi.SecurityRequirement{
			{"cookieAuth": []string{}},
			{"bearerAuth": []string{}},
		},
		Responses: map[string]*openapi.Response{
			"200": {
				Description: "Custom data",
				Content: map[string]openapi.MediaType{
					"application/json": {
						Schema: openapi.RefSchema("CustomResponse"),
					},
				},
			},
			"401": {
				Description: "Not authenticated",
				Content: map[string]openapi.MediaType{
					"application/json": {
						Schema: openapi.RefSchema("Error"),
					},
				},
			},
		},
	})
}

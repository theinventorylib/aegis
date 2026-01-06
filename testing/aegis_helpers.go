// Package testing provides shared test utilities for Aegis integration tests.
//
// This file contains Aegis-specific test helpers that depend on the main
// aegis package. It's separated to avoid import cycles.
package testing

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/theinventorylib/aegis"
	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/router"
)

// SetupTestAegis creates a configured Aegis instance for testing.
//
// This function:
//   - Sets up database connection (or skips if unavailable)
//   - Sets up Redis connection (optional)
//   - Creates Aegis with test-appropriate configuration
//   - Registers cleanup for all resources
//
// Configuration can be customized via additional options.
//
// Parameters:
//   - t: Testing instance for cleanup registration
//   - opts: Additional configuration options to apply
//
// Returns:
//   - *aegis.Aegis: Configured Aegis instance
func SetupTestAegis(t testing.TB, opts ...config.Option) *aegis.Aegis {
	t.Helper()

	cfg := DefaultTestConfig()
	db := SetupTestDB(t, cfg)
	redisClient := SetupTestRedis(t, cfg)

	if db == nil {
		t.Skip("Skipping test: database required but not available")
		return nil
	}

	// Clean database before test
	CleanDatabase(t, db)

	// Create a simple test router
	router := &testRouter{}

	// Base options with test-appropriate settings
	baseOpts := []config.Option{
		config.WithDB(db),
		config.WithRouter(router),
		config.WithSecret([]byte("test-secret-key-32-bytes-long!!")),
		config.WithAPIOnlyMode(true), // Skip CSRF for API tests
	}

	// Add Redis if available (parse from config)
	if redisClient != nil {
		// Parse Redis URL to get host and port
		redisAddr := cfg.RedisURL
		host, port := parseRedisAddr(redisAddr)
		baseOpts = append(baseOpts, config.WithRedis(host, port, "", cfg.RedisDB))
	}

	// Merge with user-provided options (user options take precedence)
	allOpts := append(baseOpts, opts...)

	// Create Aegis instance
	ctx := context.Background()
	a, err := aegis.New(ctx, allOpts...)
	if err != nil {
		t.Fatalf("Failed to create Aegis instance: %v", err)
	}

	return a
}

// SetupTestAegisWithConfig creates Aegis with custom TestConfig.
//
// This is useful when you need more control over the test infrastructure.
//
// Parameters:
//   - t: Testing instance
//   - cfg: Custom test configuration
//   - opts: Additional Aegis configuration options
//
// Returns:
//   - *aegis.Aegis: Configured Aegis instance
//   - *sql.DB: Database connection (for direct queries)
//   - *redis.Client: Redis client (may be nil)
func SetupTestAegisWithConfig(t testing.TB, cfg *TestConfig, opts ...config.Option) (*aegis.Aegis, *sql.DB, *redis.Client) {
	t.Helper()

	if cfg == nil {
		cfg = DefaultTestConfig()
	}

	db := SetupTestDB(t, cfg)
	redisClient := SetupTestRedis(t, cfg)

	if db == nil {
		t.Skip("Skipping test: database required but not available")
		return nil, nil, nil
	}

	// Clean database before test
	CleanDatabase(t, db)

	// Create a simple test router
	router := &testRouter{}

	// Base options
	baseOpts := []config.Option{
		config.WithDB(db),
		config.WithRouter(router),
		config.WithSecret([]byte("test-secret-key-32-bytes-long!!")),
		config.WithAPIOnlyMode(true),
	}

	if redisClient != nil {
		host, port := parseRedisAddr(cfg.RedisURL)
		baseOpts = append(baseOpts, config.WithRedis(host, port, "", cfg.RedisDB))
	}

	allOpts := append(baseOpts, opts...)

	ctx := context.Background()
	a, err := aegis.New(ctx, allOpts...)
	if err != nil {
		t.Fatalf("Failed to create Aegis instance: %v", err)
	}

	return a, db, redisClient
}

// testRouter is a minimal router implementation for testing.
// It satisfies the router.Router interface.
type testRouter struct {
	routes   []testRoute
	metadata []core.RouteMetadata
}

type testRoute struct {
	method  string
	pattern string
	handler http.HandlerFunc
}

func (r *testRouter) GET(path string, handler http.HandlerFunc) {
	r.routes = append(r.routes, testRoute{"GET", path, handler})
}

func (r *testRouter) POST(path string, handler http.HandlerFunc) {
	r.routes = append(r.routes, testRoute{"POST", path, handler})
}

func (r *testRouter) PUT(path string, handler http.HandlerFunc) {
	r.routes = append(r.routes, testRoute{"PUT", path, handler})
}

func (r *testRouter) PATCH(path string, handler http.HandlerFunc) {
	r.routes = append(r.routes, testRoute{"PATCH", path, handler})
}

func (r *testRouter) DELETE(path string, handler http.HandlerFunc) {
	r.routes = append(r.routes, testRoute{"DELETE", path, handler})
}

func (r *testRouter) Use(_ func(http.Handler) http.Handler) {
	// For testing, we don't need to track middleware
}

func (r *testRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Simple router implementation for testing
	for _, route := range r.routes {
		if route.method == req.Method && route.pattern == req.URL.Path {
			route.handler(w, req)
			return
		}
	}
	http.NotFound(w, req)
}

func (r *testRouter) RegisterRouteMetadata(metadata core.RouteMetadata) {
	r.metadata = append(r.metadata, metadata)
}

func (r *testRouter) GetRouteMetadata() []core.RouteMetadata {
	return r.metadata
}

// Group returns a GroupRouter for route grouping.
func (r *testRouter) Group(path string, groupName string) router.GroupRouter {
	return &testGroupRouterImpl{
		prefix:    path,
		groupName: groupName,
		parent:    r,
	}
}

// testGroupRouterImpl implements testGroupRouter for testing.
type testGroupRouterImpl struct {
	prefix    string
	groupName string
	parent    *testRouter
}

func (g *testGroupRouterImpl) GET(path string, handler http.HandlerFunc) {
	g.parent.routes = append(g.parent.routes, testRoute{"GET", g.prefix + path, handler})
}

func (g *testGroupRouterImpl) POST(path string, handler http.HandlerFunc) {
	g.parent.routes = append(g.parent.routes, testRoute{"POST", g.prefix + path, handler})
}

func (g *testGroupRouterImpl) PUT(path string, handler http.HandlerFunc) {
	g.parent.routes = append(g.parent.routes, testRoute{"PUT", g.prefix + path, handler})
}

func (g *testGroupRouterImpl) PATCH(path string, handler http.HandlerFunc) {
	g.parent.routes = append(g.parent.routes, testRoute{"PATCH", g.prefix + path, handler})
}

func (g *testGroupRouterImpl) DELETE(path string, handler http.HandlerFunc) {
	g.parent.routes = append(g.parent.routes, testRoute{"DELETE", g.prefix + path, handler})
}

func (g *testGroupRouterImpl) Use(_ func(http.Handler) http.Handler) {
	// For testing, we don't need to track middleware
}

func (g *testGroupRouterImpl) RegisterRouteMetadata(metadata core.RouteMetadata) {
	// Add group name to tags
	if g.groupName != "" {
		metadata.Tags = append([]string{g.groupName}, metadata.Tags...)
	}
	g.parent.metadata = append(g.parent.metadata, metadata)
}

// parseRedisAddr parses a Redis address string into host and port.
// Expected format: "host:port" (e.g., "localhost:6379")
func parseRedisAddr(addr string) (string, int) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return "localhost", 6379
	}

	port := 6379
	if _, err := fmt.Sscanf(parts[1], "%d", &port); err != nil {
		port = 6379
	}

	return parts[0], port
}

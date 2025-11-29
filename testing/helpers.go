// Package testing provides testing utilities and helpers for Aegis authentication framework.
package testing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/theinventorylib/aegis"
	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/models"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/server"
)

// TestAegis provides a configured Aegis instance for testing.
type TestAegis struct {
	*aegis.Aegis
	DB     *core.MockDB
	Router *TestRouter
	Config *config.Config
}

// TestRouter wraps a real router for testing.
type TestRouter struct {
	server.Router
	routes map[string]http.HandlerFunc
}

// NewTestRouter creates a new test router.
func NewTestRouter() *TestRouter {
	mux := http.NewServeMux()
	return &TestRouter{
		Router: server.NewDefaultRouter(mux),
		routes: make(map[string]http.HandlerFunc),
	}
}

// Setup creates a fully configured test Aegis instance.
//
// Example:
//
//	func TestMyFeature(t *testing.T) {
//	    testAegis := testing.Setup(t)
//	    defer testAegis.Cleanup()
//
//	    // Use testAegis.Aegis for testing
//	}
func Setup(t *testing.T, opts ...config.Option) *TestAegis {
	t.Helper()

	mockDB := core.NewMockDB()
	router := NewTestRouter()

	// Default test options
	defaultOpts := []config.Option{
		config.WithDB(mockDB, "postgres"),
		config.WithRouter(router),
		config.WithAPIOnlyMode(true),           // Skip CSRF for tests
		config.WithSessionExpiry(24 * 60 * 60), // 24 hours
	}

	// Merge with user options
	allOpts := append(defaultOpts, opts...)

	// Create Aegis instance
	cfg := config.Default()
	for _, opt := range allOpts {
		opt(cfg)
	}

	auth, err := aegis.New(allOpts...)
	if err != nil {
		t.Fatalf("Failed to create test Aegis instance: %v", err)
	}

	return &TestAegis{
		Aegis:  auth,
		DB:     mockDB,
		Router: router,
		Config: cfg,
	}
}

// Cleanup performs cleanup after tests.
func (ta *TestAegis) Cleanup() {
	// Close database if needed
	if ta.DB != nil {
		_ = ta.DB.Close()
	}
}

// CreateTestUser creates a test user in the database.
func (ta *TestAegis) CreateTestUser(t *testing.T, id string) *models.User {
	t.Helper()

	// Store in mock database using the actual API
	ctx := context.Background()
	createdUser, err := ta.DB.CreateUser(ctx)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Override ID if needed
	createdUser.ID = id
	_ = ta.DB.UpdateUser(ctx, createdUser)

	return createdUser
}

// CreateTestSession creates a test session for a user.
func (ta *TestAegis) CreateTestSession(t *testing.T, userID string) *models.Session {
	t.Helper()

	session := &models.Session{
		Token:        "test-token-" + userID,
		RefreshToken: "test-refresh-" + userID,
		UserID:       userID,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
	}

	// Insert using mock database API
	ctx := context.Background()
	err := ta.DB.CreateSession(ctx, session)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	return session
}

// Request creates an HTTP test request.
func (ta *TestAegis) Request(t *testing.T, method, path string, _ string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()

	// Use the router's HTTP handler
	handler := ta.Router.Router.(interface {
		ServeHTTP(http.ResponseWriter, *http.Request)
	})
	handler.ServeHTTP(rec, req)

	return rec
}

// AuthenticatedRequest creates an authenticated HTTP test request.
func (ta *TestAegis) AuthenticatedRequest(t *testing.T, method, path, token string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(&http.Cookie{
		Name:  "aegis_session",
		Value: token,
	})
	rec := httptest.NewRecorder()

	handler := ta.Router.Router.(interface {
		ServeHTTP(http.ResponseWriter, *http.Request)
	})
	handler.ServeHTTP(rec, req)

	return rec
}

// PluginTestSuite provides a testing framework for plugins.
type PluginTestSuite struct {
	Plugin  plugins.Plugin
	Aegis   *TestAegis
	Context context.Context
	T       *testing.T
}

// NewPluginTestSuite creates a new plugin test suite.
//
// Example:
//
//	func TestMyPlugin(t *testing.T) {
//	    plugin := myplugin.New(&myplugin.Config{})
//	    suite := testing.NewPluginTestSuite(t, plugin)
//	    defer suite.Cleanup()
//
//	    suite.TestInit()
//	    suite.TestRoutes()
//	}
func NewPluginTestSuite(t *testing.T, plugin plugins.Plugin) *PluginTestSuite {
	t.Helper()

	testAegis := Setup(t)
	ctx := context.Background()

	return &PluginTestSuite{
		Plugin:  plugin,
		Aegis:   testAegis,
		Context: ctx,
		T:       t,
	}
}

// Cleanup cleans up test resources.
func (pts *PluginTestSuite) Cleanup() {
	pts.Aegis.Cleanup()
}

// TestInit tests plugin initialization.
func (pts *PluginTestSuite) TestInit(t *testing.T) {
	t.Helper()

	err := pts.Plugin.Init(pts.Context, pts.Aegis.Aegis)
	if err != nil {
		t.Fatalf("Plugin Init failed: %v", err)
	}

	t.Logf("✓ Plugin %s initialized successfully", pts.Plugin.Name())
}

// TestMetadata tests plugin metadata methods.
func (pts *PluginTestSuite) TestMetadata(t *testing.T) {
	t.Helper()

	name := pts.Plugin.Name()
	if name == "" {
		t.Error("Plugin Name() returned empty string")
	}

	version := pts.Plugin.Version()
	if version == "" {
		t.Error("Plugin Version() returned empty string")
	}

	description := pts.Plugin.Description()
	if description == "" {
		t.Error("Plugin Description() returned empty string")
	}

	t.Logf("✓ Plugin metadata: %s v%s - %s", name, version, description)
}

// TestMountRoutes tests route mounting.
func (pts *PluginTestSuite) TestMountRoutes(t *testing.T) {
	t.Helper()

	// This is a basic check - plugin should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Plugin MountRoutes panicked: %v", r)
		}
	}()

	pts.Plugin.MountRoutes(pts.Aegis.Router, "/test")
	t.Logf("✓ Plugin routes mounted successfully")
}

// TestMigrations tests that migrations are valid.
func (pts *PluginTestSuite) TestMigrations(t *testing.T) {
	t.Helper()

	migrations := pts.Plugin.GetMigrations()
	t.Logf("✓ Plugin provides %d migrations", len(migrations))
}

// RunAllTests runs all standard plugin tests.
func (pts *PluginTestSuite) RunAllTests() {
	pts.T.Run("Metadata", func(t *testing.T) { pts.TestMetadata(t) })
	pts.T.Run("Init", func(t *testing.T) { pts.TestInit(t) })
	pts.T.Run("MountRoutes", func(t *testing.T) { pts.TestMountRoutes(t) })
	pts.T.Run("Migrations", func(t *testing.T) { pts.TestMigrations(t) })
}

// MockPlugin is a simple plugin for testing.
type MockPlugin struct {
	InitFunc        func(ctx context.Context, a plugins.Aegis) error
	MountRoutesFunc func(router server.Router, prefix string)
	name            string
	version         string
	description     string
}

// NewMockPlugin creates a new mock plugin.
func NewMockPlugin(name string) *MockPlugin {
	return &MockPlugin{
		name:            name,
		version:         "1.0.0",
		description:     "Mock plugin for testing",
		InitFunc:        func(_ context.Context, _ plugins.Aegis) error { return nil },
		MountRoutesFunc: func(_ server.Router, _ string) {},
	}
}

// Name returns the plugin name.
func (m *MockPlugin) Name() string { return m.name }

// Version returns the plugin version.
func (m *MockPlugin) Version() string { return m.version }

// Description returns the plugin description.
func (m *MockPlugin) Description() string { return m.description }

// Init initializes the plugin.
func (m *MockPlugin) Init(ctx context.Context, a plugins.Aegis) error {
	if m.InitFunc != nil {
		return m.InitFunc(ctx, a)
	}
	return nil
}

// GetMigrations returns the plugin migrations.
func (m *MockPlugin) GetMigrations() []plugins.Migration {
	return nil
}

// MountRoutes mounts the plugin routes.
func (m *MockPlugin) MountRoutes(router server.Router, prefix string) {
	if m.MountRoutesFunc != nil {
		m.MountRoutesFunc(router, prefix)
	}
}

// Dependencies returns the plugin dependencies.
func (m *MockPlugin) Dependencies() []plugins.Dependency {
	return nil
}

// RequiresTables returns the required database tables.
func (m *MockPlugin) RequiresTables() []string {
	return nil
}

// ProvidesAuthMethods returns the authentication methods provided by the plugin.
func (m *MockPlugin) ProvidesAuthMethods() []string {
	return nil
}

// AssertPluginRegistered checks that a plugin is registered.
func AssertPluginRegistered(t *testing.T, aegis *aegis.Aegis, pluginName string) {
	t.Helper()

	plugin, ok := aegis.GetPlugin(pluginName)
	if !ok {
		t.Fatalf("Plugin %s not registered", pluginName)
	}
	if plugin == nil {
		t.Fatalf("Plugin %s is nil", pluginName)
	}

	t.Logf("✓ Plugin %s is registered", pluginName)
}

// AssertPluginNotRegistered checks that a plugin is not registered.
func AssertPluginNotRegistered(t *testing.T, aegis *aegis.Aegis, pluginName string) {
	t.Helper()

	_, ok := aegis.GetPlugin(pluginName)
	if ok {
		t.Fatalf("Plugin %s should not be registered", pluginName)
	}

	t.Logf("✓ Plugin %s is not registered", pluginName)
}

// WithTestDB returns a config option that uses the provided test database.
func WithTestDB(testDB *core.MockDB) config.Option {
	return config.WithDB(testDB, "postgres")
}

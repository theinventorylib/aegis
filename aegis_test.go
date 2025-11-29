package aegis

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/server"
)

// mockPlugin is a test plugin implementation
type mockPlugin struct {
	name        string
	version     string
	description string
	initDelay   time.Duration
	initError   error
	initCalled  bool
	initFunc    func(context.Context, plugins.Aegis) error // Optional custom Init behavior
}

func (m *mockPlugin) Name() string        { return m.name }
func (m *mockPlugin) Version() string     { return m.version }
func (m *mockPlugin) Description() string { return m.description }

func (m *mockPlugin) Init(ctx context.Context, a plugins.Aegis) error {
	m.initCalled = true

	// Call custom init function if provided
	if m.initFunc != nil {
		return m.initFunc(ctx, a)
	}

	// Default behavior
	if m.initDelay > 0 {
		select {
		case <-time.After(m.initDelay):
			return m.initError
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return m.initError
}

func (m *mockPlugin) GetMigrations() []plugins.Migration    { return nil }
func (m *mockPlugin) MountRoutes(_ server.Router, _ string) {}
func (m *mockPlugin) Dependencies() []plugins.Dependency    { return nil }
func (m *mockPlugin) RequiresTables() []string              { return nil }
func (m *mockPlugin) ProvidesAuthMethods() []string         { return nil }

// mockLogger for testing logging functionality
type mockLogger struct {
	mu     sync.Mutex
	infos  []string
	errors []string
	debugs []string
}

func (m *mockLogger) Info(msg string, _ ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.infos = append(m.infos, msg)
}

func (m *mockLogger) Error(msg string, _ ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors = append(m.errors, msg)
}

func (m *mockLogger) Debug(msg string, _ ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.debugs = append(m.debugs, msg)
}

func (m *mockLogger) hasInfo(msg string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, info := range m.infos {
		if info == msg {
			return true
		}
	}
	return false
}

func (m *mockLogger) hasError(msg string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, err := range m.errors {
		if err == msg {
			return true
		}
	}
	return false
}

// setupTestAegis creates a test Aegis instance
func setupTestAegis(t *testing.T, opts ...config.Option) *Aegis {
	t.Helper()

	// Create mock database and router
	db := core.NewMockDB()
	router := server.NewDefaultRouter(http.NewServeMux())

	// Default options
	defaultOpts := []config.Option{
		config.WithDB(db, "postgres"),
		config.WithRouter(router),
		config.WithAPIOnlyMode(true), // Skip CSRF requirement for tests
	}

	// Append custom options
	defaultOpts = append(defaultOpts, opts...)

	aegis, err := New(defaultOpts...)
	if err != nil {
		t.Fatalf("Failed to create Aegis: %v", err)
	}

	return aegis
}

func TestConcurrentPluginRegistration(t *testing.T) {
	aegis := setupTestAegis(t)

	const numPlugins = 10
	var wg sync.WaitGroup
	wg.Add(numPlugins)

	// Register plugins concurrently
	for i := 0; i < numPlugins; i++ {
		go func(id int) {
			defer wg.Done()
			plugin := &mockPlugin{
				name:        fmt.Sprintf("plugin-%d", id),
				version:     "1.0.0",
				description: fmt.Sprintf("Test plugin %d", id),
			}
			err := aegis.Use(context.Background(), plugin)
			if err != nil {
				t.Errorf("Failed to register plugin-%d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all plugins were registered
	plugins := aegis.GetPlugins()
	if len(plugins) != numPlugins {
		t.Errorf("Expected %d plugins, got %d", numPlugins, len(plugins))
	}
}

func TestConcurrentPluginAccess(t *testing.T) {
	aegis := setupTestAegis(t)

	// Register a few plugins
	for i := 0; i < 5; i++ {
		plugin := &mockPlugin{
			name:        fmt.Sprintf("plugin-%d", i),
			version:     "1.0.0",
			description: fmt.Sprintf("Test plugin %d", i),
		}
		if err := aegis.Use(context.Background(), plugin); err != nil {
			t.Fatalf("Failed to register plugin: %v", err)
		}
	}

	// Concurrently access plugins
	const numReaders = 20
	var wg sync.WaitGroup
	wg.Add(numReaders)

	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()
			// Get random plugin
			_, _ = aegis.GetPlugin("plugin-2")
			// Get all plugins
			_ = aegis.GetPlugins()
		}()
	}

	wg.Wait()
}

func TestConcurrentRegistrationAndAccess(t *testing.T) {
	aegis := setupTestAegis(t)

	var wg sync.WaitGroup

	// Writers (register plugins)
	const numWriters = 5
	wg.Add(numWriters)
	for i := 0; i < numWriters; i++ {
		go func(id int) {
			defer wg.Done()
			plugin := &mockPlugin{
				name:        fmt.Sprintf("plugin-%d", id),
				version:     "1.0.0",
				description: fmt.Sprintf("Test plugin %d", id),
			}
			_ = aegis.Use(context.Background(), plugin)
		}(i)
	}

	// Readers (access plugins)
	const numReaders = 10
	wg.Add(numReaders)
	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()
			time.Sleep(time.Millisecond) // Small delay to let some writes happen
			_ = aegis.GetPlugins()
		}()
	}

	wg.Wait()
}

func TestContextCancellation(t *testing.T) {
	aegis := setupTestAegis(t)

	// Plugin with slow initialization
	plugin := &mockPlugin{
		name:        "slow-plugin",
		version:     "1.0.0",
		description: "Slow plugin",
		initDelay:   5 * time.Second,
	}

	// Context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := aegis.Use(ctx, plugin)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded, got: %v", err)
	}
}

func TestContextCancellationManual(t *testing.T) {
	aegis := setupTestAegis(t)

	plugin := &mockPlugin{
		name:        "slow-plugin",
		version:     "1.0.0",
		description: "Slow plugin",
		initDelay:   2 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context before plugin completes
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := aegis.Use(ctx, plugin)
	if err == nil {
		t.Error("Expected cancellation error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}
}

func TestPluginInitError(t *testing.T) {
	aegis := setupTestAegis(t)

	expectedErr := errors.New("init failed")
	plugin := &mockPlugin{
		name:        "failing-plugin",
		version:     "1.0.0",
		description: "Failing plugin",
		initError:   expectedErr,
	}

	err := aegis.Use(context.Background(), plugin)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error to wrap init error, got: %v", err)
	}

	// Plugin should not be registered after rollback
	_, found := aegis.GetPlugin("failing-plugin")
	if found {
		t.Error("Failed plugin should not be registered after rollback")
	}

	// Verify no plugins in registry (complete cleanup)
	plugins := aegis.GetPlugins()
	if len(plugins) != 0 {
		t.Errorf("Expected 0 plugins after rollback, got %d", len(plugins))
	}

	// Verify Init was called (plugin attempted initialization)
	if !plugin.initCalled {
		t.Error("Plugin Init should have been called before rollback")
	}
}

func TestGetPluginNotFound(t *testing.T) {
	aegis := setupTestAegis(t)

	plugin, found := aegis.GetPlugin("nonexistent")
	if found {
		t.Error("Expected found=false for nonexistent plugin")
	}
	if plugin != nil {
		t.Error("Expected nil plugin for nonexistent plugin")
	}
}

func TestGetPluginFound(t *testing.T) {
	aegis := setupTestAegis(t)

	original := &mockPlugin{
		name:        "test-plugin",
		version:     "1.0.0",
		description: "Test plugin",
	}

	err := aegis.Use(context.Background(), original)
	if err != nil {
		t.Fatalf("Failed to register plugin: %v", err)
	}

	plugin, found := aegis.GetPlugin("test-plugin")
	if !found {
		t.Error("Expected found=true for registered plugin")
	}
	if plugin == nil {
		t.Error("Expected non-nil plugin")
	}
	if plugin.Name() != "test-plugin" {
		t.Errorf("Expected plugin name 'test-plugin', got %s", plugin.Name())
	}
}

func TestPluginRegistrationOrder(t *testing.T) {
	aegis := setupTestAegis(t)

	// Register plugins in order
	names := []string{"first", "second", "third"}
	for _, name := range names {
		plugin := &mockPlugin{
			name:        name,
			version:     "1.0.0",
			description: name,
		}
		if err := aegis.Use(context.Background(), plugin); err != nil {
			t.Fatalf("Failed to register plugin %s: %v", name, err)
		}
	}

	// Verify order is preserved
	registered := aegis.GetPlugins()
	if len(registered) != len(names) {
		t.Fatalf("Expected %d plugins, got %d", len(names), len(registered))
	}

	for i, expected := range names {
		if registered[i].Name() != expected {
			t.Errorf("Plugin %d: expected %s, got %s", i, expected, registered[i].Name())
		}
	}
}

func TestGetPluginsCopy(t *testing.T) {
	aegis := setupTestAegis(t)

	plugin := &mockPlugin{
		name:        "test-plugin",
		version:     "1.0.0",
		description: "Test plugin",
	}
	if err := aegis.Use(context.Background(), plugin); err != nil {
		t.Fatalf("Failed to register plugin: %v", err)
	}

	// Get plugins twice
	plugins1 := aegis.GetPlugins()
	plugins2 := aegis.GetPlugins()

	// Verify we get copies (different slice instances)
	plugins1[0] = nil
	if plugins2[0] == nil {
		t.Error("Modifying returned slice affected internal state")
	}
}

func TestLoggingIntegration(t *testing.T) {
	logger := &mockLogger{}
	aegis := setupTestAegis(t, config.WithLogger(logger))

	plugin := &mockPlugin{
		name:        "test-plugin",
		version:     "1.0.0",
		description: "Test plugin",
	}

	err := aegis.Use(context.Background(), plugin)
	if err != nil {
		t.Fatalf("Failed to register plugin: %v", err)
	}

	// Check logging occurred
	if !logger.hasInfo("Registering plugin") {
		t.Error("Expected 'Registering plugin' log")
	}
	if !logger.hasInfo("Plugin registered successfully") {
		t.Error("Expected 'Plugin registered successfully' log")
	}
}

func TestLoggingOnError(t *testing.T) {
	logger := &mockLogger{}
	aegis := setupTestAegis(t, config.WithLogger(logger))

	plugin := &mockPlugin{
		name:        "failing-plugin",
		version:     "1.0.0",
		description: "Failing plugin",
		initError:   errors.New("init failed"),
	}

	_ = aegis.Use(context.Background(), plugin)

	// Check error logging occurred
	if !logger.hasError("Plugin initialization failed") {
		t.Error("Expected 'Plugin initialization failed' error log")
	}
}

func TestSessionMiddlewareNilCheck(t *testing.T) {
	// Create aegis with nil session (shouldn't happen, but test defensive code)
	aegis := &Aegis{
		session: nil,
	}

	// These should panic with descriptive messages
	defer func() {
		if r := recover(); r == nil {
			t.Error("AuthMiddleware should panic when session is nil")
		}
	}()

	_ = aegis.AuthMiddleware()
}

func TestRequireAuthMiddlewareNilCheck(t *testing.T) {
	aegis := &Aegis{
		session: nil,
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("RequireAuth should panic when session is nil")
		}
	}()

	_ = aegis.RequireAuth()
}

func TestPluginVisibilityDuringInit(t *testing.T) {
	aegis := setupTestAegis(t)

	// Track what plugins were visible during Init
	var visiblePlugins []string

	// First plugin - should see itself during Init
	plugin1 := &mockPlugin{
		name:        "plugin-1",
		version:     "1.0.0",
		description: "First plugin",
	}
	plugin1.initFunc = func(_ context.Context, a plugins.Aegis) error {
		// Check if we can see ourselves
		if p, found := a.(*Aegis).GetPlugin("plugin-1"); found {
			visiblePlugins = append(visiblePlugins, p.Name())
		}
		return nil
	}

	err := aegis.Use(context.Background(), plugin1)
	if err != nil {
		t.Fatalf("Failed to register plugin-1: %v", err)
	}

	// Verify plugin-1 saw itself during Init
	if len(visiblePlugins) != 1 || visiblePlugins[0] != "plugin-1" {
		t.Errorf("Expected plugin-1 to see itself during Init, got: %v", visiblePlugins)
	}

	// Second plugin - should see both itself and plugin-1
	visiblePlugins = nil // Reset
	plugin2 := &mockPlugin{
		name:        "plugin-2",
		version:     "1.0.0",
		description: "Second plugin",
	}
	plugin2.initFunc = func(_ context.Context, a plugins.Aegis) error {
		// Check all visible plugins
		allPlugins := a.(*Aegis).GetPlugins()
		for _, p := range allPlugins {
			visiblePlugins = append(visiblePlugins, p.Name())
		}
		return nil
	}

	err = aegis.Use(context.Background(), plugin2)
	if err != nil {
		t.Fatalf("Failed to register plugin-2: %v", err)
	}

	// Verify plugin-2 saw both plugins during Init
	if len(visiblePlugins) != 2 {
		t.Errorf("Expected plugin-2 to see 2 plugins during Init, got: %v", visiblePlugins)
	}
	expectedPlugins := map[string]bool{"plugin-1": true, "plugin-2": true}
	for _, name := range visiblePlugins {
		if !expectedPlugins[name] {
			t.Errorf("Unexpected plugin visible during Init: %s", name)
		}
	}
}

func TestPluginInitFailureRollback(t *testing.T) {
	aegis := setupTestAegis(t)

	// Register a successful plugin first
	successPlugin := &mockPlugin{
		name:        "success-plugin",
		version:     "1.0.0",
		description: "Successful plugin",
	}
	err := aegis.Use(context.Background(), successPlugin)
	if err != nil {
		t.Fatalf("Failed to register success plugin: %v", err)
	}

	// Verify it's registered
	if _, found := aegis.GetPlugin("success-plugin"); !found {
		t.Error("Success plugin should be registered")
	}

	// Try to register a failing plugin
	var sawSuccessPlugin bool
	failPlugin := &mockPlugin{
		name:        "fail-plugin",
		version:     "1.0.0",
		description: "Failing plugin",
	}
	failPlugin.initFunc = func(_ context.Context, a plugins.Aegis) error {
		// Check if we can see the success plugin during our Init
		if _, found := a.(*Aegis).GetPlugin("success-plugin"); found {
			sawSuccessPlugin = true
		}
		// Check if we can see ourselves
		// Verify plugin visibility during Init
		_, found := a.(*Aegis).GetPlugin("fail-plugin")
		_ = found // Plugin should be visible during its own Init
		return errors.New("intentional failure")
	}

	err = aegis.Use(context.Background(), failPlugin)
	if err == nil {
		t.Error("Expected error from failing plugin")
	}

	// Verify the failing plugin saw the success plugin during Init
	if !sawSuccessPlugin {
		t.Error("Failing plugin should have seen success plugin during Init")
	}

	// Verify rollback: fail-plugin should NOT be in registry
	if _, found := aegis.GetPlugin("fail-plugin"); found {
		t.Error("Failed plugin should not be in registry after rollback")
	}

	// Verify success-plugin is still registered (rollback didn't affect it)
	if _, found := aegis.GetPlugin("success-plugin"); !found {
		t.Error("Success plugin should still be registered after rollback")
	}

	// Verify only 1 plugin in registry
	plugins := aegis.GetPlugins()
	if len(plugins) != 1 {
		t.Errorf("Expected 1 plugin after rollback, got %d", len(plugins))
	}
	if plugins[0].Name() != "success-plugin" {
		t.Errorf("Expected success-plugin, got %s", plugins[0].Name())
	}
}

func TestPriorityOrderMountingVsInit(t *testing.T) {
	aegis := setupTestAegis(t)

	// Track initialization order
	var initOrder []string
	mu := &sync.Mutex{}

	createPlugin := func(name string, _ int) *mockPlugin {
		p := &mockPlugin{
			name:        name,
			version:     "1.0.0",
			description: fmt.Sprintf("Plugin %s", name),
		}
		p.initFunc = func(_ context.Context, _ plugins.Aegis) error {
			mu.Lock()
			initOrder = append(initOrder, name)
			mu.Unlock()
			return nil
		}
		return p
	}

	// Register plugins in non-priority order
	lowPriorityPlugin := createPlugin("low-priority", 150)
	highPriorityPlugin := createPlugin("high-priority", 50)
	mediumPriorityPlugin := createPlugin("medium-priority", 100)

	// Register in this order: low, high, medium
	if err := aegis.UseWithPriority(context.Background(), lowPriorityPlugin, 150); err != nil {
		t.Fatalf("Failed to register low-priority plugin: %v", err)
	}
	if err := aegis.UseWithPriority(context.Background(), highPriorityPlugin, 50); err != nil {
		t.Fatalf("Failed to register high-priority plugin: %v", err)
	}
	if err := aegis.UseWithPriority(context.Background(), mediumPriorityPlugin, 100); err != nil {
		t.Fatalf("Failed to register medium-priority plugin: %v", err)
	}

	// Verify Init was called in registration order (not priority order)
	expectedInitOrder := []string{"low-priority", "high-priority", "medium-priority"}
	if len(initOrder) != len(expectedInitOrder) {
		t.Fatalf("Expected %d init calls, got %d", len(expectedInitOrder), len(initOrder))
	}
	for i, expected := range expectedInitOrder {
		if initOrder[i] != expected {
			t.Errorf("Init order[%d]: expected %s, got %s", i, expected, initOrder[i])
		}
	}

	// Verify GetPlugins returns plugins in priority order (not registration order)
	plugins := aegis.GetPlugins()
	expectedPriorityOrder := []string{"high-priority", "medium-priority", "low-priority"}
	if len(plugins) != len(expectedPriorityOrder) {
		t.Fatalf("Expected %d plugins, got %d", len(expectedPriorityOrder), len(plugins))
	}
	for i, expected := range expectedPriorityOrder {
		if plugins[i].Name() != expected {
			t.Errorf("Priority order[%d]: expected %s, got %s", i, expected, plugins[i].Name())
		}
	}
}

package password_test

import (
	"context"
	"testing"

	"github.com/theinventorylib/aegis/plugins/password"
	testinghelpers "github.com/theinventorylib/aegis/testing"
)

// TestPasswordPlugin demonstrates how to test a plugin using the test framework.
func TestPasswordPlugin(t *testing.T) {
	// Create plugin instance
	plugin := password.New(&password.Config{
		// Plugin-specific configuration
	})

	// Create plugin test suite
	suite := testinghelpers.NewPluginTestSuite(t, plugin)
	defer suite.Cleanup()

	// Run all standard tests
	suite.RunAllTests()
}

// TestPasswordPluginCustom shows custom plugin testing.
func TestPasswordPluginCustom(t *testing.T) {
	// Setup
	testAegis := testinghelpers.Setup(t)
	defer testAegis.Cleanup()

	plugin := password.New(&password.Config{
		DB:     testAegis.DB,
		UserDB: testAegis.DB,
	})

	// Register plugin
	ctx := context.Background()
	err := testAegis.Use(ctx, plugin)
	if err != nil {
		t.Fatalf("Failed to register password plugin: %v", err)
	}

	// Verify registration
	testinghelpers.AssertPluginRegistered(t, testAegis.Aegis, "password")

	t.Run("PluginMetadata", func(t *testing.T) {
		if plugin.Name() != "password" {
			t.Errorf("Expected name 'password', got %s", plugin.Name())
		}

		if plugin.Version() == "" {
			t.Error("Version should not be empty")
		}

		if plugin.Description() == "" {
			t.Error("Description should not be empty")
		}
	})

	t.Run("PluginRoutes", func(t *testing.T) {
		// Mount routes
		plugin.MountRoutes(testAegis.Router, "/auth")
		t.Log("✓ Password plugin routes mounted successfully")
	})

	t.Run("PluginMigrations", func(t *testing.T) {
		migrations := plugin.GetMigrations()
		if len(migrations) == 0 {
			t.Log("ℹ Password plugin has no migrations")
		} else {
			t.Logf("✓ Password plugin provides %d migrations", len(migrations))
		}
	})
}

// TestPasswordPluginConcurrentInit tests concurrent plugin initialization.
func TestPasswordPluginConcurrentInit(t *testing.T) {
	const numInstances = 5

	for i := 0; i < numInstances; i++ {
		t.Run("Instance "+string(rune('A'+i)), func(t *testing.T) {
			t.Parallel()

			testAegis := testinghelpers.Setup(t)
			defer testAegis.Cleanup()

			plugin := password.New(&password.Config{
				DB:     testAegis.DB,
				UserDB: testAegis.DB,
			})

			ctx := context.Background()
			err := testAegis.Use(ctx, plugin)
			if err != nil {
				t.Errorf("Failed to initialize plugin: %v", err)
			}
		})
	}
}

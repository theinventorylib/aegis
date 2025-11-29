package main

import (
	"strings"
	"testing"

	"github.com/theinventorylib/aegis/plugins/admin"
	"github.com/theinventorylib/aegis/plugins/email"
	"github.com/theinventorylib/aegis/plugins/jwt"
)

func TestGetPlugins_All(t *testing.T) {
	plugins := getPlugins("all")

	if plugins == nil {
		t.Fatal("getPlugins('all') should not return nil")
	}

	if len(plugins) == 0 {
		t.Error("getPlugins('all') should return at least one plugin")
	}

	// Verify expected plugins are included
	pluginNames := make(map[string]bool)
	for _, p := range plugins {
		pluginNames[p.Name()] = true
	}

	expectedPlugins := []string{"jwt", "email", "admin"}
	for _, expected := range expectedPlugins {
		if !pluginNames[expected] {
			t.Errorf("Expected plugin '%s' not found in 'all' selection", expected)
		}
	}

	t.Logf("✓ getPlugins('all') returned %d plugins", len(plugins))
}

func TestGetPlugins_Specific(t *testing.T) {
	plugins := getPlugins("jwt,email")

	if plugins == nil {
		t.Fatal("getPlugins should not return nil for valid plugins")
	}

	if len(plugins) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(plugins))
	}

	// Verify correct plugins returned
	pluginNames := []string{}
	for _, p := range plugins {
		pluginNames = append(pluginNames, p.Name())
	}

	if !contains(pluginNames, "jwt") {
		t.Error("Expected 'jwt' plugin")
	}
	if !contains(pluginNames, "email") {
		t.Error("Expected 'email' plugin")
	}

	t.Log("✓ Specific plugin selection works correctly")
}

func TestGetPlugins_Invalid(t *testing.T) {
	plugins := getPlugins("invalid-plugin")

	if plugins != nil {
		t.Error("getPlugins should return nil for invalid plugin names")
	}

	t.Log("✓ Invalid plugin names correctly rejected")
}

func TestGetPlugins_MixedValidInvalid(t *testing.T) {
	plugins := getPlugins("jwt,invalid,email")

	if plugins != nil {
		t.Error("getPlugins should return nil if any plugin name is invalid")
	}

	t.Log("✓ Mixed valid/invalid plugin names correctly rejected")
}

func TestGetPlugins_SinglePlugin(t *testing.T) {
	plugins := getPlugins("jwt")

	if plugins == nil {
		t.Fatal("getPlugins should not return nil for valid single plugin")
	}

	if len(plugins) != 1 {
		t.Errorf("Expected 1 plugin, got %d", len(plugins))
	}

	if plugins[0].Name() != "jwt" {
		t.Errorf("Expected 'jwt' plugin, got '%s'", plugins[0].Name())
	}

	t.Log("✓ Single plugin selection works correctly")
}

func TestAvailablePlugins(t *testing.T) {
	// Verify all expected plugins can be instantiated
	expectedPlugins := map[string]interface{}{
		"email": email.New(&email.Config{}),
		"jwt":   jwt.New(&jwt.Config{}),
		"admin": admin.New(nil),
	}

	for name, plugin := range expectedPlugins {
		if plugin == nil {
			t.Errorf("Plugin '%s' failed to instantiate", name)
		}
	}

	t.Logf("✓ All %d expected plugins can be instantiated", len(expectedPlugins))
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.TrimSpace(s) == item {
			return true
		}
	}
	return false
}

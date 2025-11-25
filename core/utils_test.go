package core

import (
	"strings"
	"testing"
)

func TestGenerateID_ULID_Default(t *testing.T) {
	// ULID should be the default strategy
	if GetIDStrategy() != IDStrategyULID {
		t.Errorf("Expected default strategy to be ULID, got %s", GetIDStrategy())
	}

	// Generate multiple IDs and verify they're unique and valid ULIDs
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateID()

		// ULID should be 26 characters
		if len(id) != 26 {
			t.Errorf("Expected ULID length 26, got %d for ID: %s", len(id), id)
		}

		// Should be uppercase alphanumeric
		if strings.ToUpper(id) != id {
			t.Errorf("Expected ULID to be uppercase, got: %s", id)
		}

		// Should be unique
		if ids[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestGenerateID_UUID_Strategy(t *testing.T) {
	// Switch to UUID strategy
	SetIDStrategy(IDStrategyUUID)
	defer SetIDStrategy(IDStrategyULID) // Reset to default

	id := GenerateID()

	// UUID v4 should be 36 characters with hyphens
	if len(id) != 36 {
		t.Errorf("Expected UUID length 36, got %d for ID: %s", len(id), id)
	}

	// Should contain hyphens in correct positions (8-4-4-4-12)
	parts := strings.Split(id, "-")
	if len(parts) != 5 {
		t.Errorf("Expected UUID to have 5 parts separated by hyphens, got %d", len(parts))
	}
}

func TestGenerateID_Custom_Strategy(t *testing.T) {
	customCalled := false
	customID := "custom-id-12345"

	// Set custom generator
	SetCustomIDGenerator(func() string {
		customCalled = true
		return customID
	})
	defer SetIDStrategy(IDStrategyULID) // Reset to default

	// Generate ID
	id := GenerateID()

	// Should use custom generator
	if !customCalled {
		t.Error("Custom ID generator was not called")
	}

	if id != customID {
		t.Errorf("Expected custom ID %s, got %s", customID, id)
	}

	// Verify strategy was set to custom
	if GetIDStrategy() != IDStrategyCustom {
		t.Errorf("Expected strategy to be Custom, got %s", GetIDStrategy())
	}
}

func TestGenerateID_Uniqueness(t *testing.T) {
	// Generate many IDs quickly and ensure they're all unique
	count := 10000
	ids := make(map[string]bool, count)

	for i := 0; i < count; i++ {
		id := GenerateID()
		if ids[id] {
			t.Fatalf("Duplicate ID found at iteration %d: %s", i, id)
		}
		ids[id] = true
	}

	if len(ids) != count {
		t.Errorf("Expected %d unique IDs, got %d", count, len(ids))
	}
}

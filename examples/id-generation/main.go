package main

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/theinventorylib/aegis/core"
)

func main() {
	fmt.Println("=== Aegis ID Generation Examples ===")

	// Example 1: ULID (Default)
	fmt.Println("1. ULID (Default Strategy)")
	fmt.Println("   Benefits: Sortable, time-based, restart-safe")
	for i := 0; i < 5; i++ {
		id := core.GenerateID()
		fmt.Printf("   ID %d: %s\n", i+1, id)
		time.Sleep(1 * time.Millisecond) // Small delay to see time sorting
	}

	fmt.Println("\n2. UUID Strategy")
	fmt.Println("   Benefits: Standard format, widely compatible")
	core.SetIDStrategy(core.IDStrategyUUID)
	for i := 0; i < 3; i++ {
		id := core.GenerateID()
		fmt.Printf("   ID %d: %s\n", i+1, id)
	}

	fmt.Println("\n4. Custom Strategy")
	fmt.Println("   Benefits: Use any ID library you prefer")
	core.SetCustomIDGenerator(func() string {
		// Example: Using UUID with custom prefix
		return fmt.Sprintf("user_%s", uuid.New().String()[:8])
	})
	for i := 0; i < 3; i++ {
		id := core.GenerateID()
		fmt.Printf("   ID %d: %s\n", i+1, id)
	}

	// Reset to default for safety
	core.SetIDStrategy(core.IDStrategyULID)

	fmt.Println("\n=== Summary ===")
	fmt.Println("✅ ULID is the default (restart-safe, sortable)")
	fmt.Println("✅ UUID available for compatibility")
	fmt.Println("⚠️  Sequence only for testing")
	fmt.Println("✅ Custom generator support")
	fmt.Println("\nSee docs/id-generation.md for complete guide")
}

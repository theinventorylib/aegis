package core

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestStaticKeyManager_BasicOperations(t *testing.T) {
	ctx := context.Background()
	mgr, err := NewStaticKeyManager()
	if err != nil {
		t.Fatalf("NewStaticKeyManager() error = %v", err)
	}

	// Set and Get
	if err := mgr.Set(ctx, "key1", []byte("value1"), 0); err != nil {
		t.Errorf("Set() error = %v", err)
	}
	val, err := mgr.Get(ctx, "key1")
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if string(val) != "value1" {
		t.Errorf("Get() = %q, want %q", string(val), "value1")
	}

	// Get missing key returns error
	if _, err := mgr.Get(ctx, "missing"); err == nil {
		t.Error("Get() on missing key should return error")
	}

	// Delete
	if err := mgr.Delete(ctx, "key1"); err != nil {
		t.Errorf("Delete() error = %v", err)
	}
	if _, err := mgr.Get(ctx, "key1"); err == nil {
		t.Error("Get() after Delete() should return error")
	}

	// Delete non-existent key is a no-op
	if err := mgr.Delete(ctx, "nonexistent"); err != nil {
		t.Errorf("Delete() on missing key error = %v", err)
	}
}

// TestStaticKeyManager_ConcurrentAccess verifies there are no data races under concurrent use.
// Run with: go test -race ./core/...
func TestStaticKeyManager_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	mgr, _ := NewStaticKeyManager()

	const goroutines = 50
	const ops = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	// Concurrent writers
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < ops; j++ {
				key := "key"
				_ = mgr.Set(ctx, key, []byte("value"), time.Minute)
			}
		}()
	}

	// Concurrent readers
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < ops; j++ {
				_, _ = mgr.Get(ctx, "key")
			}
		}()
	}

	// Concurrent deleters
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < ops; j++ {
				_ = mgr.Delete(ctx, "key")
			}
		}()
	}

	wg.Wait()
}

// TestStaticKeyManager_Overwrite verifies Set overwrites existing values.
func TestStaticKeyManager_Overwrite(t *testing.T) {
	ctx := context.Background()
	mgr, _ := NewStaticKeyManager()

	_ = mgr.Set(ctx, "k", []byte("first"), 0)
	_ = mgr.Set(ctx, "k", []byte("second"), 0)

	val, err := mgr.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if string(val) != "second" {
		t.Errorf("Get() = %q, want %q", string(val), "second")
	}
}

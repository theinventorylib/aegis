// Package testing provides shared test utilities for Aegis integration tests.
//
// This package contains helper functions for setting up test infrastructure:
//   - Database setup and teardown
//   - Redis setup and teardown
//   - Aegis instance creation
//   - Test user/session creation
//
// These helpers are designed for integration tests that require real database
// and Redis connections. For unit tests, use mock implementations instead.
//
// Example usage:
//
//	func TestIntegration_UserFlow(t *testing.T) {
//		aegis := testing.SetupTestAegis(t)
//		user := testing.CreateTestUser(t, aegis, "test@example.com", "Password123!")
//
//		// Run test...
//	}
package testing

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestConfig holds configuration for test setup.
type TestConfig struct {
	// DatabaseURL is the connection string for the test database.
	// Default: uses DATABASE_URL environment variable or in-memory SQLite.
	DatabaseURL string

	// RedisURL is the connection string for test Redis.
	// Default: uses REDIS_URL environment variable or localhost:6379.
	RedisURL string

	// RedisDB is the Redis database number for tests (to isolate test data).
	// Default: 1 (separate from production DB 0)
	RedisDB int

	// SkipRedis skips Redis setup if true.
	// Useful for tests that don't require Redis.
	SkipRedis bool

	// SkipDatabase skips database setup if true.
	// Useful for pure unit tests.
	SkipDatabase bool
}

// DefaultTestConfig returns the default test configuration.
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		DatabaseURL:  getEnvOrDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/aegis_test?sslmode=disable"),
		RedisURL:     getEnvOrDefault("REDIS_URL", "localhost:6379"),
		RedisDB:      1, // Use DB 1 for tests to isolate from production
		SkipRedis:    false,
		SkipDatabase: false,
	}
}

// getEnvOrDefault returns the environment variable value or a default.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// SetupTestDB creates a test database connection.
//
// This function:
//   - Opens a database connection using the configured URL
//   - Verifies the connection is alive
//   - Registers cleanup to close the connection after the test
//
// If connection fails, the test is skipped (for CI environments without DB).
//
// Parameters:
//   - t: Testing instance for cleanup registration
//   - cfg: Optional configuration (uses DefaultTestConfig if nil)
//
// Returns:
//   - *sql.DB: Database connection (or nil if skipped)
func SetupTestDB(t testing.TB, cfg *TestConfig) *sql.DB {
	t.Helper()

	if cfg == nil {
		cfg = DefaultTestConfig()
	}

	if cfg.SkipDatabase {
		return nil
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		t.Skipf("Skipping test: failed to open database: %v", err)
		return nil
	}

	// Set connection pool settings for tests
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		t.Skipf("Skipping test: database not available: %v", err)
		return nil
	}

	// Register cleanup
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("Warning: failed to close database: %v", err)
		}
	})

	return db
}

// SetupTestRedis creates a test Redis client.
//
// This function:
//   - Creates a Redis client using the configured URL
//   - Verifies the connection is alive
//   - Flushes the test database on cleanup
//   - Registers cleanup to close the connection
//
// If connection fails, the test is skipped (for CI environments without Redis).
//
// Parameters:
//   - t: Testing instance for cleanup registration
//   - cfg: Optional configuration (uses DefaultTestConfig if nil)
//
// Returns:
//   - *redis.Client: Redis client (or nil if skipped)
func SetupTestRedis(t testing.TB, cfg *TestConfig) *redis.Client {
	t.Helper()

	if cfg == nil {
		cfg = DefaultTestConfig()
	}

	if cfg.SkipRedis {
		return nil
	}

	client := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
		DB:   cfg.RedisDB, // Use separate DB for tests
	})

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Skipf("Skipping test: Redis not available: %v", err)
		return nil
	}

	// Register cleanup
	t.Cleanup(func() {
		// Flush test database
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.FlushDB(ctx).Err(); err != nil {
			t.Logf("Warning: failed to flush Redis test DB: %v", err)
		}

		if err := client.Close(); err != nil {
			t.Logf("Warning: failed to close Redis: %v", err)
		}
	})

	return client
}

// CleanDatabase removes all test data from the database.
//
// This function truncates all Aegis tables to provide a clean state.
// It should be called before each test that requires database isolation.
//
// Parameters:
//   - t: Testing instance
//   - db: Database connection
func CleanDatabase(t testing.TB, db *sql.DB) {
	t.Helper()

	if db == nil {
		return
	}

	tables := []string{
		"aegis_verification_tokens",
		"aegis_sessions",
		"aegis_accounts",
		"aegis_users",
		// Plugin tables (if used)
		"aegis_oauth_states",
		"aegis_oauth_accounts",
		"aegis_jwt_tokens",
		"aegis_email_otps",
		"aegis_sms_otps",
		"aegis_organization_members",
		"aegis_organizations",
		"aegis_admin_users",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Disable foreign key checks temporarily for truncation
	_, _ = db.ExecContext(ctx, "SET session_replication_role = 'replica';")

	for _, table := range tables {
		_, err := db.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE;", table))
		if err != nil {
			// Table might not exist, which is fine
			t.Logf("Note: could not truncate %s: %v", table, err)
		}
	}

	// Re-enable foreign key checks
	_, _ = db.ExecContext(ctx, "SET session_replication_role = 'origin';")
}

// RunMigrations runs database migrations for testing.
//
// This function executes all Aegis migrations to set up the schema.
// It should be called once per test database setup.
//
// Parameters:
//   - t: Testing instance
//   - db: Database connection
//   - dialect: Database dialect ("postgres", "mysql", "sqlite")
func RunMigrations(t testing.TB, db *sql.DB, _ string) {
	t.Helper()

	if db == nil {
		return
	}

	// Note: This is a placeholder. In a real implementation, you would
	// import and run the actual migration functions from the auth and
	// plugin packages.
	//
	// Example:
	//   auth.RunMigrations(ctx, db, dialect)
	//   oauth.RunMigrations(ctx, db, dialect)
	//   jwt.RunMigrations(ctx, db, dialect)

	t.Log("Migrations would be run here in a full integration test setup")
}

// GenerateTestEmail generates a unique test email address.
//
// This is useful for tests that need unique email addresses to avoid
// conflicts with uniqueness constraints.
//
// Returns:
//   - string: Unique email address in format "test_<timestamp>@example.com"
func GenerateTestEmail() string {
	return fmt.Sprintf("test_%d@example.com", time.Now().UnixNano())
}

// GenerateTestPassword generates a valid test password.
//
// The password meets all common strength requirements:
//   - At least 12 characters
//   - Contains uppercase letter
//   - Contains lowercase letter
//   - Contains number
//   - Contains special character
//
// Returns:
//   - string: Valid test password
func GenerateTestPassword() string {
	return "TestP@ssw0rd123!"
}

// AssertEventually retries an assertion until it passes or times out.
//
// This is useful for testing asynchronous operations where the result
// may not be immediately available.
//
// Parameters:
//   - t: Testing instance
//   - timeout: Maximum time to wait
//   - interval: Time between retries
//   - assertion: Function that returns true when assertion passes
//   - msgAndArgs: Optional message and arguments for failure
func AssertEventually(t testing.TB, timeout, interval time.Duration, assertion func() bool, msgAndArgs ...interface{}) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if assertion() {
			return
		}
		time.Sleep(interval)
	}

	if len(msgAndArgs) > 0 {
		t.Fatalf("Assertion did not pass within %v: %v", timeout, fmt.Sprint(msgAndArgs...))
	} else {
		t.Fatalf("Assertion did not pass within %v", timeout)
	}
}

// RequireNoError fails the test immediately if err is not nil.
//
// Parameters:
//   - t: Testing instance
//   - err: Error to check
//   - msgAndArgs: Optional message and arguments for failure
func RequireNoError(t testing.TB, err error, msgAndArgs ...interface{}) {
	t.Helper()

	if err != nil {
		if len(msgAndArgs) > 0 {
			t.Fatalf("Unexpected error: %v - %v", err, fmt.Sprint(msgAndArgs...))
		} else {
			t.Fatalf("Unexpected error: %v", err)
		}
	}
}

// RequireError fails the test immediately if err is nil.
//
// Parameters:
//   - t: Testing instance
//   - err: Error to check
//   - msgAndArgs: Optional message and arguments for failure
func RequireError(t testing.TB, err error, msgAndArgs ...interface{}) {
	t.Helper()

	if err == nil {
		if len(msgAndArgs) > 0 {
			t.Fatalf("Expected error but got nil: %v", fmt.Sprint(msgAndArgs...))
		} else {
			t.Fatal("Expected error but got nil")
		}
	}
}

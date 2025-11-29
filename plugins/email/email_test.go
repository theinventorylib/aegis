package email

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/db"
	aegistesting "github.com/theinventorylib/aegis/testing"
)

type mockResult struct {
	lastInsertID int64
	rowsAffected int64
}

func (m *mockResult) LastInsertId() (int64, error) {
	return m.lastInsertID, nil
}

func (m *mockResult) RowsAffected() (int64, error) {
	return m.rowsAffected, nil
}

type mockRow struct {
	scanFunc func(dest ...interface{}) error
}

func (r *mockRow) Scan(dest ...interface{}) error {
	if r.scanFunc != nil {
		return r.scanFunc(dest...)
	}
	return nil
}

func TestEmailPlugin_Register(t *testing.T) {
	testAegis := aegistesting.Setup(t)
	defer testAegis.Cleanup()

	// Configure email plugin
	cfg := &Config{
		DB:             testAegis.DB,
		SessionService: testAegis.GetSessionService(),
	}
	plugin := New(cfg)

	// Initialize plugin
	if err := plugin.Init(context.Background(), testAegis.Aegis); err != nil {
		t.Fatalf("Failed to init plugin: %v", err)
	}

	// Mock DB ExecFunc to handle account creation and email update
	testAegis.DB.ExecFunc = func(_ context.Context, _ string, _ ...interface{}) (db.Result, error) {
		return &mockResult{rowsAffected: 1}, nil
	}

	// Mock DB QueryRowFunc to handle user lookup
	testAegis.DB.QueryRowFunc = func(_ context.Context, _ string, _ ...interface{}) db.Row {
		return &mockRow{
			scanFunc: func(dest ...interface{}) error {
				// Assume it's the user lookup query
				if len(dest) >= 1 {
					// Set ID
					if ptr, ok := dest[0].(*string); ok {
						*ptr = "test-user-id"
					}
				}
				return nil
			},
		}
	}

	// Mount routes
	plugin.MountRoutes(testAegis.Router, "/api")

	// Test registration
	reqBody := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/email/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Serve request
	testAegis.Router.ServeHTTP(rec, req)

	// Check response
	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201 Created, got %d", rec.Code)
		t.Logf("Response body: %s", rec.Body.String())
	}

	var resp core.Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success response")
	}

	// Verify user was created
	user, err := plugin.GetUserByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if user == nil {
		t.Fatal("Expected user to be returned")
	}
}

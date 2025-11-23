package core

import (
	"context"
	"testing"
)

func TestSessionService(t *testing.T) {
	// Setup
	db := NewMockDB()
	sessionConfig := DefaultSessionConfig()
	// Use static keys for testing
	sessionService := NewSessionService(db, sessionConfig)

	// Create user
	user, _ := db.CreateUser(context.Background())

	// Test CreateSession
	session, err := sessionService.CreateSession(context.Background(), user, "127.0.0.1", "UserAgent")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session.Token == "" {
		t.Error("Session token is empty")
	}

	// Test ValidateSession
	validSession, validUser, err := sessionService.ValidateSession(context.Background(), session.Token)
	if err != nil {
		t.Fatalf("Failed to validate session: %v", err)
	}

	if validSession.ID != session.ID {
		t.Error("Validated session ID mismatch")
	}
	if validUser.ID != user.ID {
		t.Error("Validated user ID mismatch")
	}

	// Test RefreshSession
	oldToken := session.Token
	refreshedSession, err := sessionService.RefreshSession(context.Background(), session.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to refresh session: %v", err)
	}

	if refreshedSession.Token == oldToken {
		t.Error("Refreshed token should be different")
	}

	// Test DeleteSession
	err = sessionService.DeleteSession(context.Background(), refreshedSession.Token)
	if err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	// Validate deleted session should fail
	_, _, err = sessionService.ValidateSession(context.Background(), refreshedSession.Token)
	if err == nil {
		t.Error("Validation should fail for deleted session")
	}
}

package core

import (
	"testing"
)

func TestPasswordHashing(t *testing.T) {
	password := "securepassword123"

	// Test hashing
	hash, err := HashPassword(password, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Test verification
	valid, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("Failed to verify password: %v", err)
	}

	if !valid {
		t.Error("Password verification failed for correct password")
	}

	// Test invalid password
	valid, err = VerifyPassword("wrongpassword", hash)
	if err != nil {
		t.Fatalf("Failed to verify invalid password: %v", err)
	}

	if valid {
		t.Error("Password verification succeeded for wrong password")
	}
}
